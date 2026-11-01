import { PanelProps } from '@grafana/data';
import { ImagePanelOptions } from 'static-image-panel/types';
import ImageRenderer from './ImageRenderer';
import React from 'react';

export default function ImagePanel(props: PanelProps<ImagePanelOptions>) {
    const { options } = props;
    return (
        <div data-testid="jaops-static-image-panel" style={{ width: '100%', height: '100%', overflow: 'hidden' }}>
            {ImageRenderer(options, options.imageUrl)}
        </div>
    );
}
