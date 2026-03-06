import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import enCommon from './locales/en/common.json';
import enLogin from './locales/en/login.json';
import enDashboard from './locales/en/dashboard.json';
import enProducts from './locales/en/products.json';
import enBuilds from './locales/en/builds.json';
import enGpg from './locales/en/gpg.json';
import enRepos from './locales/en/repos.json';
import enSettings from './locales/en/settings.json';
import enMonitors from './locales/en/monitors.json';
import enLayout from './locales/en/layout.json';

import zhCommon from './locales/zh-CN/common.json';
import zhLogin from './locales/zh-CN/login.json';
import zhDashboard from './locales/zh-CN/dashboard.json';
import zhProducts from './locales/zh-CN/products.json';
import zhBuilds from './locales/zh-CN/builds.json';
import zhGpg from './locales/zh-CN/gpg.json';
import zhRepos from './locales/zh-CN/repos.json';
import zhSettings from './locales/zh-CN/settings.json';
import zhMonitors from './locales/zh-CN/monitors.json';
import zhLayout from './locales/zh-CN/layout.json';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        common: enCommon,
        login: enLogin,
        dashboard: enDashboard,
        products: enProducts,
        builds: enBuilds,
        gpg: enGpg,
        repos: enRepos,
        settings: enSettings,
        monitors: enMonitors,
        layout: enLayout,
      },
      'zh-CN': {
        common: zhCommon,
        login: zhLogin,
        dashboard: zhDashboard,
        products: zhProducts,
        builds: zhBuilds,
        gpg: zhGpg,
        repos: zhRepos,
        settings: zhSettings,
        monitors: zhMonitors,
        layout: zhLayout,
      },
    },
    fallbackLng: 'en',
    defaultNS: 'common',
    ns: ['common', 'login', 'dashboard', 'products', 'builds', 'gpg', 'repos', 'settings', 'monitors', 'layout'],
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'i18n-language',
      caches: ['localStorage'],
      convertDetectedLanguage: (lng: string) => {
        if (lng.startsWith('zh')) return 'zh-CN';
        if (lng.startsWith('en')) return 'en';
        return lng;
      },
    },
  });

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng;
});

export default i18n;
