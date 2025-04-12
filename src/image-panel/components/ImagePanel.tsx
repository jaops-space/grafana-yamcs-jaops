import { PanelProps } from '@grafana/data';
import { PanelOptions } from 'image-panel/types';
import React from 'react';

export default function ImagePanel(props: PanelProps<PanelOptions>) {

    const { options, data } = props;
    let imageUrl = "";

    if (options.manualImageUrl) {
        imageUrl = options.imageUrl;
    } else {
        try {
            imageUrl = data.series[0].fields[0].values[0] as string;
        } catch(ignored){}
    }

    return <img src={imageUrl} alt="Image" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />;

}
