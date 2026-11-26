import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Download } from 'lucide-react'
import { useState } from 'react'
import { PageHeader } from '@/components/brandshake/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { accountApi, ApiError } from '@/lib/api'
import { useAuth } from '@/lib/auth'

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5 px-5 py-3 sm:flex-row sm:items-center sm:justify-between">
      <dt className="text-[13px] text-muted">{label}</dt>
      <dd className="text-[13px] font-medium text-fg">{value}</dd>
    </div>
  )
}

function DeleteAccount() {
  const { signOut } = useAuth()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const del = useMutation({
    mutationFn: accountApi.delete,
    onSuccess: () => {
      signOut()
      navigate({ to: '/' })
    },
  })
  const error =
    del.error instanceof ApiError
      ? (del.error.fields.password ?? del.error.message)
      : del.error
        ? 'Could not delete your account.'
        : undefined

  return (
    <>
      <Button variant="destructive" size="sm" onClick={() => setOpen(true)}>
        Delete account
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title="Delete your account?">
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            del.mutate(password)
          }}
        >
          <p className="text-sm text-muted">
            Your profile, social accounts, rates and work samples are deleted permanently. This
            can't be undone.
          </p>
          <Field label="Enter your password to confirm" error={error}>
            {(p) => (
              <Input
                {...p}
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoFocus
              />
            )}
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" size="sm" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="destructive"
              size="sm"
              loading={del.isPending}
              disabled={!password}
            >
              Delete permanently
            </Button>
          </div>
        </form>
      </Dialog>
    </>
  )
}

export function SettingsPage() {
  const { user } = useAuth()
  const exportData = useMutation({
    mutationFn: accountApi.export,
    onSuccess: (data) => {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = Object.assign(document.createElement('a'), {
        href: url,
        download: 'brandshake-data.json',
      })
      a.click()
      URL.revokeObjectURL(url)
    },
  })
  if (!user) return null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Settings" description="Your account and your data." />

      <Card>
        <CardHeader>
          <CardTitle>Account</CardTitle>
        </CardHeader>
        <dl className="divide-y divide-line">
          <Row label="Name" value={user.display_name} />
          <Row label="Email" value={user.email} />
          <Row label="Account type" value={user.role === 'creator' ? 'Creator' : 'Brand'} />
          <Row
            label="Member since"
            value={new Date(user.created_at).toLocaleDateString('en-IN', {
              day: 'numeric',
              month: 'long',
              year: 'numeric',
            })}
          />
        </dl>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>Your data</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              Download a copy of everything Brandshake stores about you.
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            loading={exportData.isPending}
            onClick={() => exportData.mutate()}
          >
            <Download aria-hidden /> Export data
          </Button>
        </CardHeader>
        {exportData.isError && (
          <p className="px-5 py-3 text-xs text-danger">Export failed. Please try again.</p>
        )}
      </Card>

      <Card className="border-danger/30">
        <CardHeader className="border-b-0">
          <div>
            <CardTitle>Delete account</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              Permanently delete your account and all of its data.
            </p>
          </div>
          <DeleteAccount />
        </CardHeader>
      </Card>
    </div>
  )
}
