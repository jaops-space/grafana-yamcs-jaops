import { PanelProps } from '@grafana/data';
import React from 'react';

export default function ImagePanel(props: PanelProps) {

    const { data } = props;
    let images: string[] = [];

    try {
        data.series.forEach((series) => {
            series.fields.forEach((field) => {
                if (field.values.length > 0 && typeof field.values[0] === 'string') {
                    images.push(field.values[field.values.length - 1] as string);
                }
            });
        });
    } catch(ignored){}

    return images.map((image, i) => 
        <img src={image} key={i} alt="Image" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />
    );

}
