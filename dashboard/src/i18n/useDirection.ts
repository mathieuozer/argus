import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';

type Direction = 'ltr' | 'rtl';

const rtlLanguages = ['ar', 'he', 'fa', 'ur'];

export function useDirection(): Direction {
  const { i18n } = useTranslation();
  const direction: Direction = rtlLanguages.includes(i18n.language) ? 'rtl' : 'ltr';

  useEffect(() => {
    document.documentElement.setAttribute('dir', direction);
    document.documentElement.setAttribute('lang', i18n.language);
  }, [direction, i18n.language]);

  return direction;
}
