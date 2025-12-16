export { defaultTheme } from './default';
export type { Theme } from './default';

import { defaultTheme } from './default';

export const availableThemes = [
  defaultTheme,
];

export const getThemeByName = (name: string) => {
  return availableThemes.find(theme => theme.name === name) || defaultTheme;
};
