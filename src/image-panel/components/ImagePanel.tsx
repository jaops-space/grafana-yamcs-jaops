import { css } from '@emotion/css';
import { GrafanaTheme2, PanelProps } from '@grafana/data';
import { Alert, Link, useStyles2 } from '@grafana/ui';
import { ROUTES } from '../../constants';
import React from 'react';
import ImageRenderer from 'static-image-panel/components/ImageRenderer';
import { ImagePanelOptions } from 'static-image-panel/types';
import { prefixRoute } from 'utils/utils.routing';

const getStyles = (theme: GrafanaTheme2) => ({
    emptyState: css`
        padding: ${theme.spacing(2)};
    `,
});

export default function ImagePanel(props: PanelProps<ImagePanelOptions>) {
    const styles = useStyles2(getStyles);
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
    } catch (ignored) {}

    return (
        <div data-testid="jaops-telemetric-image-panel">
            {images.length === 0 ? (
                <div className={styles.emptyState}>
                    <Alert severity="info" title="No image received">
                        Select an Image query and wait for an image parameter value.
                        <br />
                        <Link href={prefixRoute(ROUTES.Image)} color="primary">
                            Open Image Panel setup
                        </Link>
                    </Alert>
                </div>
            ) : (
                images.map((image) => ImageRenderer(options, image))
            )}
        </div>
    );
}
