import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import React from 'react';

const getStyles = (theme: GrafanaTheme2) => ({
    section: (separated: boolean) => css({
        width: '100%',
        paddingTop: separated ? theme.spacing(1.25) : 0,
        borderTop: separated ? `1px solid ${theme.colors.border.weak}` : undefined,
    }),
    header: css({
        display: 'flex',
        flexDirection: 'column',
        gap: theme.spacing(0.25),
        marginBottom: theme.spacing(1),
    }),
    title: css({
        margin: 0,
        fontSize: theme.typography.bodySmall.fontSize,
        fontWeight: theme.typography.fontWeightMedium,
    }),
    description: css({
        color: theme.colors.text.secondary,
        fontSize: theme.typography.bodySmall.fontSize,
        lineHeight: theme.typography.bodySmall.lineHeight,
    }),
    fields: (columns?: number) => css({
        display: 'grid',
        gridTemplateColumns: columns ? `repeat(${columns}, minmax(0, 1fr))` : 'repeat(auto-fit, minmax(180px, 1fr))',
        gap: `${theme.spacing(1)} ${theme.spacing(1.5)}`,
        alignItems: 'start',
    }),
});

export function FormSection(props: {
    title?: string;
    description?: string;
    children: React.ReactNode;
    columns?: number;
    separated?: boolean;
}) {
    const { title, description, children, columns, separated = true } = props;
    const styles = useStyles2(getStyles);

    return (
        <section className={styles.section(separated)}>
            {(title || description) && (
                <div className={styles.header}>
                    {title && <h5 className={styles.title}>{title}</h5>}
                    {description && <span className={styles.description}>{description}</span>}
                </div>
            )}
            <div className={styles.fields(columns)}>{children}</div>
        </section>
    );
}
