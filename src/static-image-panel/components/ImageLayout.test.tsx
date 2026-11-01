import React from 'react';
import { render, screen } from '@testing-library/react';
import { PanelProps } from '@grafana/data';
import StaticImagePanel from './StaticImagePanel';
import TelemetricImagePanel from '../../image-panel/components/ImagePanel';
import { ImagePanelOptions } from '../types';

jest.mock('@grafana/runtime', () => ({
    getTemplateSrv: () => ({ replace: (value: string) => value }),
}));
jest.mock('@grafana/ui', () => ({
    Alert: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    Link: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
    useStyles2: () => ({ emptyState: '' }),
}));

const imageURL = 'http://localhost:8092/frame.svg';

describe('image panel containment', () => {
    it.each([
        ['static', StaticImagePanel, 'jaops-static-image-panel'],
        ['telemetric', TelemetricImagePanel, 'jaops-telemetric-image-panel'],
    ] as const)('%s uses the panel height rather than the image intrinsic height', (_name, Component, marker) => {
        const props = {
            options: { imageUrl: imageURL, objectFit: 'contain' },
            data: { series: [{ fields: [{ name: 'image', values: [imageURL] }] }] },
        } as PanelProps<ImagePanelOptions>;
        render(<Component {...props} />);
        expect(screen.getByTestId(marker)).toHaveStyle({ width: '100%', height: '100%', overflow: 'hidden' });
        expect(screen.getByRole('img')).toHaveStyle({ display: 'block', objectFit: 'contain', height: '100%' });
        expect(screen.getByRole('img')).toHaveAttribute('src', imageURL);
    });

    it('keeps multiple telemetric images in bounded grid cells', () => {
        const urls = [1, 2, 3, 4].map((number) => `http://localhost:8092/frame-${number}.svg`);
        const props = {
            options: { objectFit: 'contain' },
            data: { series: [{ fields: urls.map((url, index) => ({ name: `image-${index}`, values: [url] })) }] },
        } as PanelProps<ImagePanelOptions>;
        render(<TelemetricImagePanel {...props} />);
        expect(screen.getAllByRole('img').map((image) => image.getAttribute('src'))).toEqual(urls);
        expect(screen.getByTestId('jaops-telemetric-image-panel')).toHaveStyle({
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gridAutoRows: 'minmax(0, 1fr)',
        });
    });
});
