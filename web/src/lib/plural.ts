/**
 * Принимает: число и три формы слова (1 трата, 2 траты, 5 трат).
 * Делает: выбирает форму по правилам русского языка (11–14 всегда «много»).
 * Возвращает: «12 трат».
 */
export function plural(n: number, forms: [string, string, string]): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  const form =
    mod10 === 1 && mod100 !== 11
      ? forms[0]
      : mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)
        ? forms[1]
        : forms[2];
  return `${n} ${form}`;
}
