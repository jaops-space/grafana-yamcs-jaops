import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { PluginPage } from '@grafana/runtime';
import { Card, Alert, Text, useStyles2 } from '@grafana/ui';
import React from 'react';
import CommandingImage from '../img/how-to-use/commanding.png';

const getStyles = (theme: GrafanaTheme2) => ({
    image: css`
        display: block;
        margin: ${theme.spacing(2.5)} auto;
    `,
});

function CommandingPanelSetup() {
    const styles = useStyles2(getStyles);

    return (
        <PluginPage>
            <p>
                Welcome! This guide will help you set up and use the Yamcs Commanding Panel in Grafana. Follow the steps
                below to get started.
            </p>

            <Card>
                <Card.Heading>Step 1: Create a New Panel</Card.Heading>
                <Card.Description>
                    Go to your Grafana dashboard and click on <Text color="info">Add Panel</Text>. Then, set the data
                    source to <Text color="primary">Yamcs Datasource</Text>.
                </Card.Description>
            </Card>

            <img src={CommandingImage} width={1200} className={styles.image} />

            <Card>
                <Card.Heading>
                    <span data-testid="jaops-setup-page-commanding">Step 2: Choose Commanding Query Type</span>
                </Card.Heading>
                <Card.Description>
                    In the panel{"'"}s query settings, select <Text color="primary">Commanding</Text> as the query
                    type. Then, search for your desired command.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 3: Configure the Commanding Panel</Card.Heading>
                <Card.Description>
                    Once you{"'"}ve selected your command, choose <Text color="primary">Yamcs Commanding Panel</Text>{' '}
                    as the visualization type. You will be presented with a form to configure the button and appearance
                    of your panel.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 4: Review Before Sending</Card.Heading>
                <Card.Description>
                    Check the selected endpoint, command name, arguments, and comment before pressing the command
                    button. Use dual-button mode for paired on/off commands where the current state should stay visible.
                </Card.Description>
            </Card>

            <Alert title="Tip" severity="info">
                You can configure each button with specific arguments and comments for the command, making the panel
                fully customizable and easy to use.
            </Alert>
        </PluginPage>
    );
}

export default CommandingPanelSetup;
