import { PanelPlugin } from '@grafana/data';
import { PanelOptions } from './types';
import ImagePanel from './components/ImagePanel';

export const plugin = new PanelPlugin<PanelOptions>(ImagePanel)
    .setPanelOptions((builder) => {
        builder.addBooleanSwitch({
            name: "Manual Image URL",
            description: "Use manual image URL instead of data source",
            defaultValue: true,
            category: ['Image Source'],
            path: 'manualImageUrl'
        })
        .addTextInput({
            name: "Image URL",
            description: "URL of the image to display (make sure to remove any data stream queries for the image to be displayed)",
            category: ['Image Source'],
            path: 'imageUrl',
            showIf: (options) => options.manualImageUrl === true,
        })
    });
