import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/field'
import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { AuthLayout, FormError } from './auth-layout'

const schema = z.object({
  email: z.email('Enter a valid email address.'),
  password: z.string().min(1, 'Enter your password.'),
})
type Values = z.infer<typeof schema>

export function LoginPage() {
  const { signIn } = useAuth()
  const navigate = useNavigate()
  const form = useForm<Values>({ resolver: zodResolver(schema) })

  const login = useMutation({
    mutationFn: api.login,
    onSuccess: (session) => {
      signIn(session)
      navigate({ to: '/dashboard' })
    },
  })

  const errors = form.formState.errors
  const serverError =
    login.error instanceof ApiError
      ? login.error.message
      : login.error
        ? 'Sign-in failed.'
        : undefined

  return (
    <AuthLayout
      title="Sign in"
      subtitle="Welcome back to Brandshake."
      footer={
        <>
          New here?{' '}
          <Link to="/signup" className="font-medium text-fg underline-offset-4 hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((v) => login.mutate(v))}
        noValidate
      >
        <FormError message={serverError} />
        <Field label="Email" error={errors.email?.message}>
          {(p) => <Input {...p} type="email" autoComplete="email" {...form.register('email')} />}
        </Field>
        <Field label="Password" error={errors.password?.message}>
          {(p) => (
            <Input
              {...p}
              type="password"
              autoComplete="current-password"
              {...form.register('password')}
            />
          )}
        </Field>
        <Button type="submit" size="lg" loading={login.isPending} className="mt-2">
          Sign in
        </Button>
      </form>
    </AuthLayout>
  )
}
