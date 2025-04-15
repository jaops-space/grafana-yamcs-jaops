import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { ROUTES } from '../../constants';
import CommandingPanelSetup from 'pages/Commanding';
import ImagePanelSetup from 'pages/Image';
import Overview from 'pages/Overview';
const HowToUse = React.lazy(() => import('../../pages/HowToUse'));

function App(props: AppRootProps) {
    return (
        <Routes>
            <Route path={ROUTES.HowToUse} element={<HowToUse />} />
            <Route path={ROUTES.Commanding} element={<CommandingPanelSetup />} />
            <Route path={ROUTES.Image} element={<ImagePanelSetup />} />

            {/* Default page */}
            <Route path="*" element={<Overview />} />
        </Routes>
    );
}

export default App;
