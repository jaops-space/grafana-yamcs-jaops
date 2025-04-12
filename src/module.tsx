import { AppPlugin, type AppRootProps } from '@grafana/data';
import { LoadingPlaceholder } from '@grafana/ui';
import React, { Suspense, lazy } from 'react';

const LazyApp = lazy(() => import('./components/App/App'));

const App = (props: AppRootProps) => (
    <Suspense fallback={<LoadingPlaceholder text="" />}>
        <LazyApp {...props} />
    </Suspense>
);

export const plugin = new AppPlugin<{}>().setRootPage(App);
