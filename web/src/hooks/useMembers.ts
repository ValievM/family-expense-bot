import { fetchMembers } from '../api/client';
import type { Member } from '../api/types';
import { useFetch, type FetchState } from './useFetch';

/**
 * Принимает: ничего.
 * Делает: загружает участников семьи; reload нужен после создания расхода для нового ID.
 * Возвращает: FetchState со списком участников.
 */
export function useMembers(): FetchState<Member[]> {
  return useFetch(fetchMembers, []);
}
