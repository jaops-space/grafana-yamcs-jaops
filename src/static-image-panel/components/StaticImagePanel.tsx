import { PanelProps } from '@grafana/data';
import { PanelOptions } from 'static-image-panel/types';
import React from 'react';

export default function ImagePanel(props: PanelProps<PanelOptions>) {

    const { options } = props;
    let image = options.imageUrl;

    return <img src={image} alt="Image" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />

}
