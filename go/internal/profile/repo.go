package profile

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/store"
)

var (
	ErrNotFound    = errors.New("profile not found")
	ErrHandleTaken = errors.New("handle already taken")
)

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

const profileColumns = `p.handle, p.headline, p.bio, p.city, p.state, p.website,
	p.niches, p.languages, p.open_to_collabs, p.industry, p.updated_at`

func scanProfile(row pgx.Row, extra ...any) (Profile, error) {
	var p Profile
	dest := append([]any{
		&p.Handle, &p.Headline, &p.Bio, &p.City, &p.State, &p.Website,
		&p.Niches, &p.Languages, &p.OpenToCollabs, &p.Industry, &p.UpdatedAt,
	}, extra...)
	err := row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

func (r *Repo) Get(ctx context.Context, userID string) (Profile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx,
		`SELECT `+profileColumns+` FROM profiles p WHERE p.user_id = $1`, userID))
	p.UserID = userID
	return p, err
}

// Upsert creates or updates the caller's profile.
func (r *Repo) Upsert(ctx context.Context, userID string, in Profile) (Profile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `
		INSERT INTO profiles AS p (user_id, handle, headline, bio, city, state, website,
			niches, languages, open_to_collabs, industry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			handle = EXCLUDED.handle, headline = EXCLUDED.headline, bio = EXCLUDED.bio,
			city = EXCLUDED.city, state = EXCLUDED.state, website = EXCLUDED.website,
			niches = EXCLUDED.niches, languages = EXCLUDED.languages,
			open_to_collabs = EXCLUDED.open_to_collabs, industry = EXCLUDED.industry,
			updated_at = now()
		RETURNING `+profileColumns,
		userID, in.Handle, in.Headline, in.Bio, in.City, in.State, in.Website,
		nonNil(in.Niches), nonNil(in.Languages), in.OpenToCollabs, in.Industry))
	if store.IsUniqueViolation(err, "profiles_handle_key") {
		return Profile{}, ErrHandleTaken
	}
	p.UserID = userID
	return p, err
}

// HandleAvailable reports whether handle is free (or already owned by userID).
func (r *Repo) HandleAvailable(ctx context.Context, handle, userID string) (bool, error) {
	var taken bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM profiles WHERE handle = $1 AND user_id::text <> $2)`,
		handle, userID).Scan(&taken)
	return !taken, err
}

func (r *Repo) ReplaceSocials(ctx context.Context, userID string, in []SocialAccount) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM social_accounts WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for i, s := range in {
			if _, err := tx.Exec(ctx,
				`INSERT INTO social_accounts (user_id, platform, handle, url, followers, position)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				userID, s.Platform, s.Handle, s.URL, s.Followers, i); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repo) ReplaceRates(ctx context.Context, userID string, in []Rate) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM rate_cards WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for i, rt := range in {
			if _, err := tx.Exec(ctx,
				`INSERT INTO rate_cards (user_id, position, deliverable, price_minor, note)
				 VALUES ($1, $2, $3, $4, $5)`,
				userID, i, rt.Deliverable, rt.PriceMinor, rt.Note); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repo) ReplaceSamples(ctx context.Context, userID string, in []Sample) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM work_samples WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for i, s := range in {
			if _, err := tx.Exec(ctx,
				`INSERT INTO work_samples (user_id, position, title, url, platform, brand_name)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				userID, i, s.Title, s.URL, s.Platform, s.BrandName); err != nil {
				return err
			}
		}
		return nil
	})
}

// PublicByHandle loads everything shown on a public profile page.
func (r *Repo) PublicByHandle(ctx context.Context, handle string) (Public, error) {
	var pub Public
	p, err := scanProfile(r.db.QueryRow(ctx, `
		SELECT `+profileColumns+`, p.user_id::text, u.role, u.display_name, u.created_at
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.handle = $1`, handle),
		&pub.UserID, &pub.Role, &pub.DisplayName, &pub.MemberSince)
	if err != nil {
		return Public{}, err
	}
	uid := pub.UserID
	pub.Profile = p
	pub.UserID = uid
	return pub, r.loadLists(ctx, &pub)
}

// Lists loads the socials, rates and samples for userID.
func (r *Repo) Lists(ctx context.Context, userID string) (socials []SocialAccount, rates []Rate, samples []Sample, err error) {
	pub := Public{Profile: Profile{UserID: userID}}
	err = r.loadLists(ctx, &pub)
	return pub.Socials, pub.Rates, pub.Samples, err
}

func (r *Repo) loadLists(ctx context.Context, pub *Public) error {
	var err error
	rows, _ := r.db.Query(ctx,
		`SELECT platform, handle, url, followers, verified FROM social_accounts
		 WHERE user_id = $1 ORDER BY position`, pub.UserID)
	if pub.Socials, err = pgx.CollectRows(rows, pgx.RowToStructByPos[SocialAccount]); err != nil {
		return err
	}
	rows, _ = r.db.Query(ctx,
		`SELECT deliverable, price_minor, note FROM rate_cards WHERE user_id = $1 ORDER BY position`, pub.UserID)
	if pub.Rates, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Rate]); err != nil {
		return err
	}
	rows, _ = r.db.Query(ctx,
		`SELECT title, url, platform, brand_name FROM work_samples WHERE user_id = $1 ORDER BY position`, pub.UserID)
	pub.Samples, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Sample])
	return err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
