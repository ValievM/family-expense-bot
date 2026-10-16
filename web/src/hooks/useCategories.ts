import { fetchCategories } from '../api/client';
import type { Category } from '../api/types';
import { useFetch, type FetchState } from './useFetch';

export function useCategories(): FetchState<Category[]> {
  return useFetch(fetchCategories, []);
}
