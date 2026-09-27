import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"

export function useRules() {
  return useQuery({
    queryKey: ["rules"],
    queryFn: api.rules,
    staleTime: Infinity,
  }).data
}
