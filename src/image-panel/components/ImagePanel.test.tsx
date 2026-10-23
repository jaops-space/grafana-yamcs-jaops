import React from 'react';
import { render, screen } from '@testing-library/react';
import { dateTime, EventBusSrv, FieldType, LoadingState, PanelProps, toDataFrame } from '@grafana/data';
import { ImagePanelOptions } from 'static-image-panel/types';
import ImagePanel from './ImagePanel';

jest.mock('static-image-panel/components/ImageRenderer', () => ({
    __esModule: true,
    default: (_options: ImagePanelOptions, url: string) => <img key={url} src={url} alt="Telemetry" />,
}));

jest.mock('@grafana/ui', () => ({
    Alert: ({ children, title }: { children: React.ReactNode; title: string }) => (
        <div>
            {title}
            {children}
        </div>
    ),
    Link: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
    Icon: () => <span aria-hidden="true" />,
    useStyles2: () => ({ emptyState: 'empty' }),
}));

jest.mock('utils/utils.routing', () => ({ prefixRoute: (path: string) => path }));

function props(urls: string[]): PanelProps<ImagePanelOptions> {
    const timeRange = { from: dateTime(0), to: dateTime(1), raw: { from: 'now-1m', to: 'now' } };
    return {
        id: 1,
        width: 448,
        height: 210,
        transparent: false,
        renderCounter: 0,
        title: 'Camera',
        options: { manualImageUrl: false, imageUrl: '', objectFit: 'contain' },
        data: {
            state: LoadingState.Done,
            series: urls.map((url) =>
                toDataFrame({
                    fields: [{ name: 'Image', type: FieldType.string, values: [url] }],
                })
            ),
            timeRange,
        },
        timeRange,
        timeZone: 'browser',
        fieldConfig: { defaults: {}, overrides: [] },
        eventBus: new EventBusSrv(),
        onOptionsChange: jest.fn(),
        onFieldConfigChange: jest.fn(),
        onChangeTimeRange: jest.fn(),
        replaceVariables: (value: string) => value,
    };
}

it('constrains the image layout to Grafana panel dimensions', () => {
    render(<ImagePanel {...props(['http://localhost/image.png'])} />);
    expect(screen.getByTestId('jaops-telemetric-image-panel')).toHaveStyle({
        width: '448px',
        height: '210px',
        display: 'grid',
        gridTemplateColumns: 'repeat(1, minmax(0, 1fr))',
        gridTemplateRows: 'minmax(0, 1fr)',
    });
    expect(screen.getByRole('img')).toHaveAttribute('src', 'http://localhost/image.png');
});

it('allocates bounded columns for multiple images and resizes with the panel', () => {
    const initial = props(['http://localhost/a.png', 'http://localhost/b.png']);
    const { rerender } = render(<ImagePanel {...initial} />);
    expect(screen.getAllByRole('img')).toHaveLength(2);
    expect(screen.getByTestId('jaops-telemetric-image-panel')).toHaveStyle({
        gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
    });
    rerender(<ImagePanel {...initial} width={320} height={160} />);
    expect(screen.getByTestId('jaops-telemetric-image-panel')).toHaveStyle({ width: '320px', height: '160px' });
});

it('keeps the explicit missing-image message', () => {
    render(<ImagePanel {...props([])} />);
    expect(screen.getByText(/No image received/)).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
});

it('shows an optional presentation waiting state without setup instructions', () => {
    const initial = props([]);
    render(<ImagePanel {...initial} options={{ ...initial.options, emptyStateText: 'Awaiting monitoring feed' }} />);
    expect(screen.getByRole('status')).toHaveTextContent('Awaiting monitoring feed');
    expect(screen.queryByText(/Open Image Panel setup/)).not.toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
});

it('shows telemetry instead of the waiting state when an image arrives', () => {
    const initial = props([]);
    const options = { ...initial.options, emptyStateText: 'Awaiting capture' };
    const { rerender } = render(<ImagePanel {...initial} options={options} />);
    rerender(<ImagePanel {...props(['http://localhost/capture.png'])} options={options} />);
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.getByRole('img')).toHaveAttribute('src', 'http://localhost/capture.png');
});

it('retains setup guidance for a blank custom waiting message', () => {
    const initial = props([]);
    render(<ImagePanel {...initial} options={{ ...initial.options, emptyStateText: '  ' }} />);
    expect(screen.getByText(/No image received/)).toBeInTheDocument();
});
