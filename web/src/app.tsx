import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AuthProvider, useAuth } from '@/lib/auth'
import { router } from '@/router'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false },
    mutations: { retry: false },
  },
})

function RoutedApp() {
  const auth = useAuth()
  useEffect(() => {
    router.invalidate()
  }, [auth.user])
  return <RouterProvider router={router} context={{ auth }} />
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RoutedApp />
      </AuthProvider>
    </QueryClientProvider>
  )
}
