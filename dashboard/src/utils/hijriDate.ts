/**
 * Hijri (Islamic) calendar conversion utilities.
 * Uses the Umm al-Qura calendar approximation algorithm.
 */

interface HijriDate {
  year: number;
  month: number;
  day: number;
}

const HIJRI_MONTHS = [
  'Muharram', 'Safar', 'Rabi\' al-Awwal', 'Rabi\' al-Thani',
  'Jumada al-Ula', 'Jumada al-Thani', 'Rajab', 'Sha\'ban',
  'Ramadan', 'Shawwal', 'Dhu al-Qi\'dah', 'Dhu al-Hijjah'
];

const HIJRI_MONTHS_AR = [
  'محرم', 'صفر', 'ربيع الأول', 'ربيع الثاني',
  'جمادى الأولى', 'جمادى الآخرة', 'رجب', 'شعبان',
  'رمضان', 'شوال', 'ذو القعدة', 'ذو الحجة'
];

/**
 * Convert Gregorian date to Hijri date (approximate Umm al-Qura algorithm).
 */
export function toHijri(date: Date): HijriDate {
  const jd = gregorianToJD(date.getFullYear(), date.getMonth() + 1, date.getDate());
  return jdToHijri(jd);
}

/**
 * Format a date as dual Gregorian/Hijri string.
 */
export function formatDualDate(date: Date, locale: string = 'en'): string {
  const hijri = toHijri(date);
  const gregStr = date.toLocaleDateString(locale === 'ar' ? 'ar-SA' : 'en-US', {
    year: 'numeric', month: 'short', day: 'numeric'
  });

  const months = locale === 'ar' ? HIJRI_MONTHS_AR : HIJRI_MONTHS;
  const hijriStr = `${hijri.day} ${months[hijri.month - 1]} ${hijri.year}`;

  return `${gregStr} / ${hijriStr}`;
}

/**
 * Format just the Hijri date.
 */
export function formatHijriDate(date: Date, locale: string = 'en'): string {
  const hijri = toHijri(date);
  const months = locale === 'ar' ? HIJRI_MONTHS_AR : HIJRI_MONTHS;
  return `${hijri.day} ${months[hijri.month - 1]} ${hijri.year}`;
}

/**
 * Get Hijri month name.
 */
export function getHijriMonthName(month: number, locale: string = 'en'): string {
  const months = locale === 'ar' ? HIJRI_MONTHS_AR : HIJRI_MONTHS;
  return months[month - 1] || '';
}

// Julian Day Number from Gregorian date
function gregorianToJD(year: number, month: number, day: number): number {
  if (month <= 2) {
    year -= 1;
    month += 12;
  }
  const A = Math.floor(year / 100);
  const B = 2 - A + Math.floor(A / 4);
  return Math.floor(365.25 * (year + 4716)) + Math.floor(30.6001 * (month + 1)) + day + B - 1524.5;
}

// Hijri date from Julian Day Number (Kuwaiti algorithm)
function jdToHijri(jd: number): HijriDate {
  const l = Math.floor(jd) - 1948440 + 10632;
  const n = Math.floor((l - 1) / 10631);
  const remainder = l - 10631 * n + 354;
  const j = Math.floor((10985 - remainder) / 5316) * Math.floor((50 * remainder) / 17719) +
            Math.floor(remainder / 5670) * Math.floor((43 * remainder) / 15238);
  const adjustedL = remainder - Math.floor((30 - j) / 15) * Math.floor((17719 * j) / 50) -
                    Math.floor(j / 16) * Math.floor((15238 * j) / 43) + 29;
  const month = Math.floor((24 * adjustedL) / 709);
  const day = adjustedL - Math.floor((709 * month) / 24);
  const year = 30 * n + j - 30;

  return { year, month, day };
}

export default { toHijri, formatDualDate, formatHijriDate, getHijriMonthName };
