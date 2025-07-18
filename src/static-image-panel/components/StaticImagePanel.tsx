import { PanelProps } from '@grafana/data';
import { ImagePanelOptions } from 'static-image-panel/types';
import ImageRenderer from './ImageRenderer';

export default function ImagePanel(props: PanelProps<ImagePanelOptions>) {
    const { options } = props;
    return ImageRenderer(options, options.imageUrl);
}
