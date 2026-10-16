import { fetchMembers } from '../api/client';
import type { Member } from '../api/types';
import { useFetch, type FetchState } from './useFetch';

export function useMembers(): FetchState<Member[]> {
  return useFetch(fetchMembers, []);
}
