import { PanelProps } from '@grafana/data';
import ImageRenderer from 'static-image-panel/components/ImageRenderer';
import { ImagePanelOptions } from 'static-image-panel/types';

export default function ImagePanel(props: PanelProps<ImagePanelOptions>) {

    const { options, data } = props;
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

    return images.map((image) => ImageRenderer(options, image));

}
