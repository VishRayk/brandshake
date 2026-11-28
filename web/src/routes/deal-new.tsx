import { useMutation } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, ClipboardPaste, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import { DealForm, toFormValues, type DealFormValues } from '@/components/deals/deal-form'
import { PageHeader } from '@/components/brandshake/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/controls'
import { ApiError, dealApi, type DealDraftResponse, type DealInput } from '@/lib/api'
import { cn } from '@/lib/cn'
import { useDealMutation } from '@/lib/queries'

type Mode = 'paste' | 'manual'

function DraftNotes({ draft }: { draft: DealDraftResponse }) {
  const items = [
    ...draft.risk_flags.map((f) => ({
      key: `r-${f.term}-${f.message}`,
      text: f.message,
      high: f.severity === 'high',
    })),
    ...draft.open_questions.map((q) => ({ key: `q-${q}`, text: q, high: false })),
  ]
  return (
    <Card className="border-info/30 bg-info-subtle/40">
      <div className="px-5 py-4">
        <p className="text-[13px] font-medium text-fg">
          Drafted from your message. Check every field before saving.
        </p>
        {draft.summary && <p className="mt-1 text-[13px] text-muted">{draft.summary}</p>}
        {items.length > 0 && (
          <ul className="mt-3 flex flex-col gap-1.5">
            {items.map((i) => (
              <li key={i.key} className="flex items-start gap-2 text-[13px] text-fg">
                <TriangleAlert
                  className={cn(
                    'mt-0.5 size-3.5 shrink-0',
                    i.high ? 'text-danger' : 'text-warning',
                  )}
                  aria-hidden
                />
                {i.text}
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}

export function NewDealPage() {
  const navigate = useNavigate()
  const [mode, setMode] = useState<Mode>('paste')
  const [text, setText] = useState('')
  const [draft, setDraft] = useState<DealDraftResponse | null>(null)
  const [initial, setInitial] = useState<DealFormValues>(() => toFormValues(null))
  const [formKey, setFormKey] = useState(0)

  const drafting = useMutation({
    mutationFn: dealApi.draft,
    onSuccess: (r) => {
      setDraft(r)
      setInitial(toFormValues(r.suggestion))
      setFormKey((k) => k + 1)
    },
  })
  const create = useDealMutation((d: DealInput) => dealApi.create(d))

  const draftError =
    drafting.error instanceof ApiError
      ? (drafting.error.fields.text ?? drafting.error.message)
      : drafting.error
        ? 'Drafting failed. Fill in the deal manually.'
        : undefined

  const save = async (input: DealInput) => {
    const status = draft ? draft.suggestion.status : 'lead'
    const source = draft ? 'ai_draft' : 'manual'
    const { deal } = await create.mutateAsync({ ...input, status, source })
    navigate({ to: '/deals/$dealId', params: { dealId: deal.id } })
  }

  const showForm = mode === 'manual' || draft !== null

  return (
    <div className="flex flex-col gap-6">
      <Link
        to="/deals"
        className="flex items-center gap-1.5 self-start text-[13px] text-muted hover:text-fg"
      >
        <ArrowLeft className="size-4" aria-hidden /> Deals
      </Link>
      <PageHeader
        title="New deal"
        description="Start from a message you received, or fill it in yourself."
      />

      <div
        className="flex rounded-md border border-line-strong p-0.5 self-start"
        role="tablist"
        aria-label="How to start"
      >
        {(
          [
            ['paste', 'Paste email or chat'],
            ['manual', 'Enter manually'],
          ] as const
        ).map(([m, label]) => (
          <button
            key={m}
            type="button"
            role="tab"
            aria-selected={mode === m}
            onClick={() => setMode(m)}
            className={cn(
              'h-8 rounded-sm px-3 text-[13px] font-medium transition-colors duration-150',
              mode === m ? 'bg-fg text-background' : 'text-muted hover:text-fg',
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {mode === 'paste' && !draft && (
        <Card>
          <CardHeader>
            <div>
              <CardTitle>Paste the message</CardTitle>
              <p className="mt-0.5 text-xs text-muted">
                An email, Instagram DM or WhatsApp chat about the collaboration. Nothing is saved
                until you confirm.
              </p>
            </div>
          </CardHeader>
          <div className="flex flex-col gap-3 px-5 py-5">
            <Textarea
              aria-label="Message"
              rows={10}
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={
                'Hi Maya! We’d love 2 Instagram reels for our protein bar launch, live by 10 Dec. Our budget is ₹15,000 with 1 revision…'
              }
            />
            {draftError && (
              <p role="alert" className="text-[13px] text-danger">
                {draftError}{' '}
                <button
                  type="button"
                  className="font-medium underline"
                  onClick={() => setMode('manual')}
                >
                  Enter manually
                </button>
              </p>
            )}
            <Button
              type="button"
              className="self-start"
              loading={drafting.isPending}
              disabled={text.trim().length === 0}
              onClick={() => drafting.mutate(text)}
            >
              <ClipboardPaste aria-hidden /> Draft deal
            </Button>
          </div>
        </Card>
      )}

      {mode === 'paste' && draft && (
        <div className="flex flex-col gap-3">
          <DraftNotes draft={draft} />
          <button
            type="button"
            className="self-start text-[13px] font-medium text-muted underline-offset-4 hover:text-fg hover:underline"
            onClick={() => {
              setDraft(null)
              setInitial(toFormValues(null))
            }}
          >
            Paste a different message
          </button>
        </div>
      )}

      {showForm && (
        <DealForm
          key={`${mode}-${formKey}`}
          initial={mode === 'manual' && !draft ? toFormValues(null) : initial}
          submitLabel="Save deal"
          onSubmit={save}
          onCancel={() => navigate({ to: '/deals' })}
        />
      )}
    </div>
  )
}
