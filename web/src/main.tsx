// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import 'dockview-react/dist/styles/dockview.css';
import './styles.css';
import { initI18n } from './i18n';
import { applyTheme, useUi } from './state/ui';
import { App } from './App';

const i18n = initI18n();
document.documentElement.lang = i18n.language;
applyTheme(useUi.getState().theme);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
