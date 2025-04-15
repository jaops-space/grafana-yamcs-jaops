import { AppPlugin, PluginExtensionPoints, type AppRootProps } from '@grafana/data';
import { LoadingPlaceholder } from '@grafana/ui';
import ImagePanelSetup from 'pages/Image';
import React, { Suspense, lazy } from 'react';

const LazyApp = lazy(() => import('./components/App/App'));

const App = (props: AppRootProps) => (
    <Suspense fallback={<LoadingPlaceholder text="" />}>
        <LazyApp {...props} />
    </Suspense>
);

export const plugin = new AppPlugin<{}>().setRootPage(App)
    .addLink({
        title: 'How to Use the plugin',
        icon: 'question-circle',
        path: '/how-to-use',
        targets: [PluginExtensionPoints.DashboardPanelMenu]
    })
    .addLink({
        title: 'Commanding Panel Setup',
        icon: 'rocket',
        path: '/commanding-setup',
        targets: [PluginExtensionPoints.DashboardPanelMenu]
    })
    .addConfigPage({
        title: 'Image Panel Setup',
        icon: 'gf-portrait',
        body: ImagePanelSetup,
        id: 'image-panel-setup',
    });
