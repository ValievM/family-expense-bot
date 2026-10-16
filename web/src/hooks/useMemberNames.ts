import { useCallback, useState } from 'react';
import { loadNames, saveNames, withName, type NameMap } from '../lib/members';
import { browserStorage } from '../lib/storage';

/**
 * Принимает: ничего.
 * Делает: хранит локальные имена членов семьи в localStorage; без хранилища имена живут до перезагрузки.
 * Возвращает: [имена, переименовать(id, имя)].
 */
export function useMemberNames(): [NameMap, (id: number, name: string) => void] {
  const [names, setNames] = useState<NameMap>(() => loadNames(browserStorage()));

  const rename = useCallback(
    (id: number, name: string) => {
      const next = withName(names, id, name);
      saveNames(browserStorage(), next);
      setNames(next);
    },
    [names],
  );

  return [names, rename];
}
