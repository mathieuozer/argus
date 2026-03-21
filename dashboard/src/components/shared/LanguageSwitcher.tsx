import { useTranslation } from 'react-i18next';

export default function LanguageSwitcher() {
  const { i18n } = useTranslation();

  const toggleLanguage = () => {
    const newLang = i18n.language === 'ar' ? 'en' : 'ar';
    i18n.changeLanguage(newLang);
  };

  return (
    <button
      className="language-switcher"
      onClick={toggleLanguage}
      title={i18n.language === 'ar' ? 'Switch to English' : '\u0627\u0644\u062A\u0628\u062F\u064A\u0644 \u0625\u0644\u0649 \u0627\u0644\u0639\u0631\u0628\u064A\u0629'}
    >
      {i18n.language === 'ar' ? 'EN' : '\u0639\u0631'}
    </button>
  );
}
