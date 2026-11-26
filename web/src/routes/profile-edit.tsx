import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Check, ExternalLink, Plus, Trash2, X } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { PageHeader } from '@/components/brandshake/page-header'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { ChipGroup, SaveStatus, Select, Switch, Textarea } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { profileApi, type MyProfile, type Profile } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { paiseToRupeesText, rupeesToPaise } from '@/lib/format'
import { useMyProfile, useProfileMutation } from '@/lib/queries'
import { applyServerErrors } from '@/lib/server-errors'
import { INDUSTRIES, LANGUAGES, NICHES, PLATFORMS, STATES } from '@/lib/vocab'

const HANDLE_RE = /^[a-z0-9][a-z0-9_.]{1,28}[a-z0-9]$/
const optionalUrl = z
  .string()
  .trim()
  .refine(
    (v) => v === '' || /^https?:\/\/[^\s/]+\.[^\s]+$/.test(v),
    'Enter a full link starting with https://',
  )

function Section({
  title,
  description,
  children,
  footer,
}: {
  title: string
  description: string
  children: ReactNode
  footer: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <div>
          <CardTitle>{title}</CardTitle>
          <p className="mt-0.5 text-xs text-muted">{description}</p>
        </div>
      </CardHeader>
      <div className="flex flex-col gap-4 px-5 py-5">{children}</div>
      <div className="flex items-center justify-end gap-3 border-t border-line bg-surface px-5 py-3">
        {footer}
      </div>
    </Card>
  )
}

function saveState(status: 'idle' | 'pending' | 'success' | 'error', isDirty: boolean) {
  if (status === 'pending') return 'saving' as const
  if (status === 'success' && !isDirty) return 'saved' as const
  if (status === 'error') return 'error' as const
  return 'idle' as const
}

/* ---------- Basics ---------- */

const basicsSchema = z.object({
  handle: z
    .string()
    .trim()
    .transform((v) => v.replace(/^@/, '').toLowerCase())
    .refine(
      (v) => HANDLE_RE.test(v) && !v.includes('..'),
      'Use 3–30 lowercase letters, numbers, dots or underscores.',
    ),
  headline: z.string().trim().max(120, 'Keep it under 120 characters.'),
  bio: z.string().trim().max(1000, 'Keep it under 1,000 characters.'),
  city: z.string().trim().max(60),
  state: z.string(),
  website: optionalUrl,
  niches: z.array(z.string()).max(5),
  languages: z.array(z.string()).max(6),
  open_to_collabs: z.boolean(),
  industry: z.string(),
})
type BasicsIn = z.input<typeof basicsSchema>
type BasicsOut = z.output<typeof basicsSchema>

function toBasics(p: Profile | null): BasicsIn {
  return {
    handle: p?.handle ?? '',
    headline: p?.headline ?? '',
    bio: p?.bio ?? '',
    city: p?.city ?? '',
    state: p?.state ?? '',
    website: p?.website ?? '',
    niches: p?.niches ?? [],
    languages: p?.languages ?? [],
    open_to_collabs: p?.open_to_collabs ?? true,
    industry: p?.industry ?? '',
  }
}

function useDebounced<T>(value: T, ms = 400): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms)
    return () => clearTimeout(t)
  }, [value, ms])
  return v
}

function HandleHint({ handle, saved }: { handle: string; saved?: string }) {
  const h = useDebounced(handle.trim().replace(/^@/, '').toLowerCase())
  const enabled = HANDLE_RE.test(h) && h !== saved
  const { data, isFetching } = useQuery({
    queryKey: ['handle', h],
    queryFn: () => profileApi.checkHandle(h),
    enabled,
    staleTime: 10_000,
  })
  if (!enabled || isFetching || !data) return null
  return data.available ? (
    <p className="flex items-center gap-1 text-xs text-success">
      <Check className="size-3.5" aria-hidden /> {window.location.host}/@{h} is available
    </p>
  ) : (
    <p className="flex items-center gap-1 text-xs text-danger">
      <X className="size-3.5" aria-hidden /> {data.reason}
    </p>
  )
}

function BasicsForm({ data, role }: { data: MyProfile; role: 'creator' | 'brand' }) {
  const form = useForm<BasicsIn, unknown, BasicsOut>({
    resolver: zodResolver(basicsSchema),
    defaultValues: toBasics(data.profile),
  })
  const [formError, setFormError] = useState<string>()
  const save = useProfileMutation(profileApi.save, (prev, r) => ({ ...prev, profile: r.profile }))
  const handle = useWatch({ control: form.control, name: 'handle' })
  const errors = form.formState.errors

  const onSubmit = form.handleSubmit((v) => {
    setFormError(undefined)
    save.mutate(v, {
      onSuccess: (r) => form.reset(toBasics(r.profile)),
      onError: (e) => setFormError(applyServerErrors(form, e)),
    })
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <Section
        title="Basics"
        description={
          role === 'creator'
            ? 'How brands find and recognise you.'
            : 'How creators recognise your brand.'
        }
        footer={
          <>
            {formError && <span className="mr-auto text-xs text-danger">{formError}</span>}
            <SaveStatus state={saveState(save.status, form.formState.isDirty)} />
            <Button
              type="submit"
              size="sm"
              loading={save.isPending}
              disabled={!form.formState.isDirty && !!data.profile}
            >
              Save
            </Button>
          </>
        }
      >
        <Field
          label="Handle"
          hint={`Your public link: ${window.location.host}/@handle`}
          error={errors.handle?.message}
        >
          {(p) => (
            <div className="flex flex-col gap-1.5">
              <div className="flex">
                <span className="flex h-9 items-center rounded-l-md border border-r-0 border-line-strong bg-surface px-3 text-sm text-subtle">
                  @
                </span>
                <Input
                  {...p}
                  className="rounded-l-none"
                  autoComplete="off"
                  {...form.register('handle')}
                />
              </div>
              <HandleHint handle={handle} saved={data.profile?.handle} />
            </div>
          )}
        </Field>
        <Field
          label="Headline"
          hint={
            role === 'creator'
              ? 'e.g. "Fitness & food creator from Bengaluru"'
              : 'e.g. "Clean protein snacks, made in India"'
          }
          error={errors.headline?.message}
        >
          {(p) => <Input {...p} {...form.register('headline')} />}
        </Field>
        <Field label="About" error={errors.bio?.message}>
          {(p) => <Textarea {...p} rows={4} {...form.register('bio')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="City" error={errors.city?.message}>
            {(p) => <Input {...p} autoComplete="address-level2" {...form.register('city')} />}
          </Field>
          <Field label="State" error={errors.state?.message}>
            {(p) => (
              <Select {...p} {...form.register('state')}>
                <option value="">Select a state</option>
                {STATES.map((s) => (
                  <option key={s}>{s}</option>
                ))}
              </Select>
            )}
          </Field>
        </div>
        <Field label="Website" error={errors.website?.message}>
          {(p) => <Input {...p} type="url" placeholder="https://" {...form.register('website')} />}
        </Field>

        {role === 'creator' ? (
          <>
            <div className="flex flex-col gap-1.5">
              <p className="text-[13px] font-medium text-fg">Niches</p>
              <Controller
                control={form.control}
                name="niches"
                render={({ field }) => (
                  <ChipGroup
                    label="Niches"
                    options={NICHES}
                    value={field.value}
                    onChange={field.onChange}
                    max={5}
                  />
                )}
              />
              <p className="text-xs text-subtle">Pick up to 5.</p>
            </div>
            <div className="flex flex-col gap-1.5">
              <p className="text-[13px] font-medium text-fg">Languages you create in</p>
              <Controller
                control={form.control}
                name="languages"
                render={({ field }) => (
                  <ChipGroup
                    label="Languages"
                    options={LANGUAGES}
                    value={field.value}
                    onChange={field.onChange}
                    max={6}
                  />
                )}
              />
            </div>
            <Controller
              control={form.control}
              name="open_to_collabs"
              render={({ field }) => (
                <Switch
                  checked={field.value}
                  onChange={field.onChange}
                  label="Open to collaborations"
                  description="Show brands that you're taking new partnerships."
                />
              )}
            />
          </>
        ) : (
          <Field label="Industry" error={errors.industry?.message}>
            {(p) => (
              <Select {...p} {...form.register('industry')}>
                <option value="">Select an industry</option>
                {INDUSTRIES.map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        )}
      </Section>
    </form>
  )
}

/* ---------- List editors ---------- */

function RowShell({
  children,
  onRemove,
  label,
}: {
  children: ReactNode
  onRemove: () => void
  label: string
}) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-line bg-surface p-3">
      <div className="grid flex-1 gap-3 sm:grid-cols-2">{children}</div>
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove ${label}`}
        className="rounded-md p-1.5 text-subtle transition-colors duration-150 hover:bg-danger-subtle hover:text-danger"
      >
        <Trash2 className="size-4" aria-hidden />
      </button>
    </div>
  )
}

const socialsSchema = z.object({
  socials: z.array(
    z.object({
      platform: z.string().min(1, 'Choose a platform.'),
      handle: z.string().trim().min(1, 'Enter your handle.').max(60),
      url: z
        .string()
        .trim()
        .regex(/^https?:\/\/\S+$/, 'Paste the link to your profile.'),
      followers: z.coerce
        .number<string | number>()
        .int()
        .min(0, 'Enter a number.')
        .max(1_000_000_000),
    }),
  ),
})
type SocialsIn = z.input<typeof socialsSchema>
type SocialsOut = z.output<typeof socialsSchema>

function SocialsForm({ data }: { data: MyProfile }) {
  const toForm = (d: MyProfile): SocialsIn => ({
    socials: d.socials.map(({ platform, handle, url, followers }) => ({
      platform,
      handle,
      url,
      followers,
    })),
  })
  const form = useForm<SocialsIn, unknown, SocialsOut>({
    resolver: zodResolver(socialsSchema),
    defaultValues: toForm(data),
  })
  const rows = useFieldArray({ control: form.control, name: 'socials' })
  const [formError, setFormError] = useState<string>()
  const save = useProfileMutation(profileApi.saveSocials, (prev, r) => ({
    ...prev,
    socials: r.socials,
  }))
  const used = useWatch({ control: form.control, name: 'socials' }).map((s) => s.platform)
  const nextPlatform = PLATFORMS.find(([p]) => !used.includes(p))?.[0]
  const errors = form.formState.errors.socials

  const onSubmit = form.handleSubmit((v) => {
    setFormError(undefined)
    save.mutate(v.socials, {
      onSuccess: (r) => form.reset({ socials: r.socials }),
      onError: (e) => setFormError(applyServerErrors(form, e)),
    })
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <Section
        title="Social accounts"
        description="Link the accounts brands should look at. Follower counts are shown as you enter them."
        footer={
          <>
            {formError && <span className="mr-auto text-xs text-danger">{formError}</span>}
            <SaveStatus state={saveState(save.status, form.formState.isDirty)} />
            <Button
              type="submit"
              size="sm"
              loading={save.isPending}
              disabled={!form.formState.isDirty}
            >
              Save
            </Button>
          </>
        }
      >
        {rows.fields.map((row, i) => (
          <RowShell key={row.id} onRemove={() => rows.remove(i)} label="account">
            <Field label="Platform" error={errors?.[i]?.platform?.message}>
              {(p) => (
                <Select {...p} {...form.register(`socials.${i}.platform`)}>
                  {PLATFORMS.map(([v, label]) => (
                    <option key={v} value={v} disabled={v !== used[i] && used.includes(v)}>
                      {label}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
            <Field label="Handle" error={errors?.[i]?.handle?.message}>
              {(p) => (
                <Input {...p} placeholder="@yourhandle" {...form.register(`socials.${i}.handle`)} />
              )}
            </Field>
            <Field label="Profile link" error={errors?.[i]?.url?.message}>
              {(p) => (
                <Input
                  {...p}
                  type="url"
                  placeholder="https://"
                  {...form.register(`socials.${i}.url`)}
                />
              )}
            </Field>
            <Field label="Followers" error={errors?.[i]?.followers?.message}>
              {(p) => (
                <Input {...p} inputMode="numeric" {...form.register(`socials.${i}.followers`)} />
              )}
            </Field>
          </RowShell>
        ))}
        {nextPlatform ? (
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() =>
              rows.append({ platform: nextPlatform, handle: '', url: '', followers: 0 })
            }
          >
            <Plus aria-hidden /> Add account
          </Button>
        ) : (
          <p className="text-xs text-subtle">You've added every supported platform.</p>
        )}
      </Section>
    </form>
  )
}

const ratesSchema = z.object({
  rates: z
    .array(
      z.object({
        deliverable: z.string().trim().min(1, 'Describe the deliverable.').max(80),
        price: z
          .string()
          .refine((v) => rupeesToPaise(v) !== null, 'Enter an amount in rupees, e.g. 15000.'),
        note: z.string().trim().max(120),
      }),
    )
    .max(10),
})
type RatesForm = z.infer<typeof ratesSchema>

function RatesForm({ data }: { data: MyProfile }) {
  const toForm = (d: Pick<MyProfile, 'rates'>): RatesForm => ({
    rates: d.rates.map((r) => ({
      deliverable: r.deliverable,
      price: paiseToRupeesText(r.price_minor),
      note: r.note,
    })),
  })
  const form = useForm<RatesForm>({
    resolver: zodResolver(ratesSchema),
    defaultValues: toForm(data),
  })
  const rows = useFieldArray({ control: form.control, name: 'rates' })
  const [formError, setFormError] = useState<string>()
  const save = useProfileMutation(profileApi.saveRates, (prev, r) => ({ ...prev, rates: r.rates }))
  const errors = form.formState.errors.rates

  const onSubmit = form.handleSubmit((v) => {
    setFormError(undefined)
    const rates = v.rates.map((r) => ({
      deliverable: r.deliverable,
      note: r.note,
      price_minor: rupeesToPaise(r.price)!,
    }))
    save.mutate(rates, {
      onSuccess: (r) => form.reset(toForm(r)),
      onError: (e) => setFormError(applyServerErrors(form, e, { price_minor: 'price' })),
    })
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <Section
        title="Rate card"
        description="Your starting prices. Brands see these on your profile; you can still negotiate each deal."
        footer={
          <>
            {formError && <span className="mr-auto text-xs text-danger">{formError}</span>}
            <SaveStatus state={saveState(save.status, form.formState.isDirty)} />
            <Button
              type="submit"
              size="sm"
              loading={save.isPending}
              disabled={!form.formState.isDirty}
            >
              Save
            </Button>
          </>
        }
      >
        {rows.fields.map((row, i) => (
          <RowShell key={row.id} onRemove={() => rows.remove(i)} label="rate">
            <Field label="Deliverable" error={errors?.[i]?.deliverable?.message}>
              {(p) => (
                <Input
                  {...p}
                  placeholder="Instagram Reel"
                  {...form.register(`rates.${i}.deliverable`)}
                />
              )}
            </Field>
            <Field label="Price (₹)" error={errors?.[i]?.price?.message}>
              {(p) => (
                <Input
                  {...p}
                  inputMode="decimal"
                  placeholder="15000"
                  className="tabular-nums"
                  {...form.register(`rates.${i}.price`)}
                />
              )}
            </Field>
            <div className="sm:col-span-2">
              <Field label="Note (optional)" error={errors?.[i]?.note?.message}>
                {(p) => (
                  <Input
                    {...p}
                    placeholder="Includes 1 revision"
                    {...form.register(`rates.${i}.note`)}
                  />
                )}
              </Field>
            </div>
          </RowShell>
        ))}
        {rows.fields.length < 10 && (
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() => rows.append({ deliverable: '', price: '', note: '' })}
          >
            <Plus aria-hidden /> Add rate
          </Button>
        )}
      </Section>
    </form>
  )
}

const samplesSchema = z.object({
  samples: z
    .array(
      z.object({
        title: z.string().trim().min(1, 'Give this a title.').max(100),
        url: z
          .string()
          .trim()
          .regex(/^https?:\/\/\S+$/, 'Paste a full link starting with https://'),
        brand_name: z.string().trim().max(80),
      }),
    )
    .max(12),
})
type SamplesForm = z.infer<typeof samplesSchema>

function SamplesForm({ data }: { data: MyProfile }) {
  const toForm = (d: Pick<MyProfile, 'samples'>): SamplesForm => ({
    samples: d.samples.map(({ title, url, brand_name }) => ({ title, url, brand_name })),
  })
  const form = useForm<SamplesForm>({
    resolver: zodResolver(samplesSchema),
    defaultValues: toForm(data),
  })
  const rows = useFieldArray({ control: form.control, name: 'samples' })
  const [formError, setFormError] = useState<string>()
  const save = useProfileMutation(profileApi.saveSamples, (prev, r) => ({
    ...prev,
    samples: r.samples,
  }))
  const errors = form.formState.errors.samples

  const onSubmit = form.handleSubmit((v) => {
    setFormError(undefined)
    save.mutate(v.samples, {
      onSuccess: (r) => form.reset(toForm(r)),
      onError: (e) => setFormError(applyServerErrors(form, e)),
    })
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <Section
        title="Work samples"
        description="Links to your best posts or videos, especially past brand collaborations."
        footer={
          <>
            {formError && <span className="mr-auto text-xs text-danger">{formError}</span>}
            <SaveStatus state={saveState(save.status, form.formState.isDirty)} />
            <Button
              type="submit"
              size="sm"
              loading={save.isPending}
              disabled={!form.formState.isDirty}
            >
              Save
            </Button>
          </>
        }
      >
        {rows.fields.map((row, i) => (
          <RowShell key={row.id} onRemove={() => rows.remove(i)} label="sample">
            <Field label="Title" error={errors?.[i]?.title?.message}>
              {(p) => (
                <Input
                  {...p}
                  placeholder="Protein bar launch Reel"
                  {...form.register(`samples.${i}.title`)}
                />
              )}
            </Field>
            <Field label="Brand (optional)" error={errors?.[i]?.brand_name?.message}>
              {(p) => (
                <Input {...p} placeholder="FitBar" {...form.register(`samples.${i}.brand_name`)} />
              )}
            </Field>
            <div className="sm:col-span-2">
              <Field label="Link" error={errors?.[i]?.url?.message}>
                {(p) => (
                  <Input
                    {...p}
                    type="url"
                    placeholder="https://www.instagram.com/reel/…"
                    {...form.register(`samples.${i}.url`)}
                  />
                )}
              </Field>
            </div>
          </RowShell>
        ))}
        {rows.fields.length < 12 && (
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() => rows.append({ title: '', url: '', brand_name: '' })}
          >
            <Plus aria-hidden /> Add sample
          </Button>
        )}
      </Section>
    </form>
  )
}

/* ---------- Page ---------- */

export function ProfileEditPage() {
  const { user } = useAuth()
  const { data, isPending, isError, refetch } = useMyProfile()
  if (!user) return null

  const handle = data?.profile?.handle

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Profile"
        description={
          user.role === 'creator'
            ? 'Your media kit, shared as a single link.'
            : 'Your brand page on Brandshake.'
        }
        actions={
          handle && (
            <Link
              to="/@{$handle}"
              params={{ handle }}
              className={buttonVariants({ variant: 'secondary', size: 'sm' })}
            >
              <ExternalLink aria-hidden /> View public profile
            </Link>
          )
        }
      />

      {isPending && (
        <div className="flex flex-col gap-6" aria-busy>
          {[0, 1].map((i) => (
            <div key={i} className="h-64 animate-pulse rounded-md border border-line bg-raised" />
          ))}
        </div>
      )}

      {isError && (
        <Card className="px-5 py-6 text-sm">
          <p className="text-fg">Your profile couldn't be loaded.</p>
          <Button variant="secondary" size="sm" className="mt-3" onClick={() => refetch()}>
            Try again
          </Button>
        </Card>
      )}

      {data && (
        <>
          <BasicsForm data={data} role={user.role} />
          {data.profile ? (
            <>
              <SocialsForm data={data} />
              {user.role === 'creator' && (
                <>
                  <RatesForm data={data} />
                  <SamplesForm data={data} />
                </>
              )}
            </>
          ) : (
            <p className="text-sm text-muted">
              Save your basics to add social accounts
              {user.role === 'creator' ? ', rates and work samples' : ''}.
            </p>
          )}
        </>
      )}
    </div>
  )
}
