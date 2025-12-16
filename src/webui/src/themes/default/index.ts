export interface Theme {
  name: string;
  displayName: string;
  cssPath: string;
}

export const defaultTheme: Theme = {
  name: 'default',
  displayName: 'Default',
  cssPath: './theme.css',
};

export default defaultTheme;
