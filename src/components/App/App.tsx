import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { ROUTES } from '../../constants';
const HowToUse = React.lazy(() => import('../../pages/HowToUse'));

function App(props: AppRootProps) {
    return (
        <Routes>
            <Route path={ROUTES.HowToUse} element={<HowToUse />} />

            {/* Default page */}
            <Route path="*" element={<HowToUse />} />
        </Routes>
    );
}

export default App;
