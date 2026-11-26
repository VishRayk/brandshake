import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { profileApi, type MyProfile } from './api'

export const keys = {
  myProfile: ['me', 'profile'] as const,
  publicProfile: (handle: string) => ['profiles', handle] as const,
}

export function useMyProfile() {
  return useQuery({ queryKey: keys.myProfile, queryFn: profileApi.mine })
}

/** A mutation that writes one part of the caller's profile and patches the cache. */
export function useProfileMutation<TInput, TResult>(
  fn: (input: TInput) => Promise<TResult>,
  apply: (prev: MyProfile, result: TResult) => MyProfile,
) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (result) => {
      qc.setQueryData<MyProfile>(keys.myProfile, (prev) =>
        apply(prev ?? { profile: null, socials: [], rates: [], samples: [] }, result),
      )
      qc.invalidateQueries({ queryKey: ['profiles'] })
    },
  })
}
