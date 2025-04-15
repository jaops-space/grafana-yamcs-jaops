import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from './types';
import StaticImagePanel from './components/StaticImagePanel';

export const plugin = new PanelPlugin<PanelOptions>(StaticImagePanel)
    .setPanelOptions((builder) => {
        builder.addTextInput({
            name: "Image URL",
            description: "URL of the image to display",
            category: ['Image Source'],
            path: 'imageUrl',
        })
    });
