import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/field'
import { api, ApiError, type Role } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { cn } from '@/lib/cn'
import { AuthLayout, FormError } from './auth-layout'

const schema = z.object({
  role: z.enum(['creator', 'brand']),
  display_name: z.string().trim().min(1, 'Enter a name.').max(80, 'Keep it under 80 characters.'),
  email: z.email('Enter a valid email address.'),
  password: z.string().min(8, 'Use at least 8 characters.').max(128),
})
type Values = z.infer<typeof schema>

const roles: { value: Role; title: string; body: string }[] = [
  { value: 'creator', title: 'Creator', body: 'Get paid for sponsored content' },
  { value: 'brand', title: 'Brand', body: 'Hire creators with confidence' },
]

export function SignupPage({ initialRole }: { initialRole?: Role }) {
  const { signIn } = useAuth()
  const navigate = useNavigate()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { role: initialRole ?? 'creator' },
  })
  const role = useWatch({ control: form.control, name: 'role' })

  const signup = useMutation({
    mutationFn: api.signup,
    onSuccess: (session) => {
      signIn(session)
      navigate({ to: '/dashboard' })
    },
  })

  // Map server-side validation errors onto fields.
  useEffect(() => {
    const err = signup.error
    if (err instanceof ApiError) {
      for (const [field, message] of Object.entries(err.fields)) {
        form.setError(field as keyof Values, { message })
      }
      if (err.code === 'email_taken') form.setError('email', { message: err.message })
    }
  }, [signup.error, form])

  const errors = form.formState.errors
  const serverError =
    signup.error instanceof ApiError &&
    !Object.keys(signup.error.fields).length &&
    signup.error.code !== 'email_taken'
      ? signup.error.message
      : undefined

  return (
    <AuthLayout
      title="Create your account"
      subtitle="Free for creators. Brands pay a fee only on funded pacts."
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="font-medium text-fg underline-offset-4 hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((v) => signup.mutate(v))}
        noValidate
      >
        <FormError message={serverError} />

        <fieldset>
          <legend className="text-[13px] font-medium text-fg">I'm joining as</legend>
          <div className="mt-1.5 grid grid-cols-2 gap-2">
            {roles.map((r) => (
              <label
                key={r.value}
                className={cn(
                  'cursor-pointer rounded-md border px-3 py-2.5 transition-colors duration-150',
                  'has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-ring',
                  role === r.value ? 'border-fg bg-raised' : 'border-line-strong hover:bg-surface',
                )}
              >
                <input
                  type="radio"
                  value={r.value}
                  className="sr-only"
                  {...form.register('role')}
                />
                <span className="block text-sm font-medium text-fg">{r.title}</span>
                <span className="block text-xs text-muted">{r.body}</span>
              </label>
            ))}
          </div>
        </fieldset>

        <Field
          label={role === 'brand' ? 'Company name' : 'Your name'}
          error={errors.display_name?.message}
        >
          {(p) => (
            <Input
              {...p}
              autoComplete={role === 'brand' ? 'organization' : 'name'}
              {...form.register('display_name')}
            />
          )}
        </Field>
        <Field label="Email" error={errors.email?.message}>
          {(p) => <Input {...p} type="email" autoComplete="email" {...form.register('email')} />}
        </Field>
        <Field label="Password" hint="At least 8 characters." error={errors.password?.message}>
          {(p) => (
            <Input
              {...p}
              type="password"
              autoComplete="new-password"
              {...form.register('password')}
            />
          )}
        </Field>
        <Button type="submit" size="lg" loading={signup.isPending} className="mt-2">
          Create account
        </Button>
      </form>
    </AuthLayout>
  )
}
