import { fetchCategories } from '../api/client';
import type { Category } from '../api/types';
import { useFetch, type FetchState } from './useFetch';

/**
 * Принимает: ничего.
 * Делает: один раз загружает список категорий.
 * Возвращает: FetchState со списком категорий.
 */
export function useCategories(): FetchState<Category[]> {
  return useFetch(fetchCategories, []);
}
