import { PluginPage } from '@grafana/runtime';
import { Card, Link } from '@grafana/ui';
import React from 'react';
import { prefixRoute } from 'utils/utils.routing';

function Overview() {
    return (
        <PluginPage>
            <p>Welcome! This section will guide you through the setup and usage of the Yamcs Datasource and Panels in Grafana.</p>

            <Card>
                <Card.Heading>How to Use the Yamcs Plugin</Card.Heading>
                <Card.Description>
                    <p>
                        This guide will help you set up and use the Yamcs Datasource and Panels in Grafana. 
                        Learn about adding the data source, configuring panels, and using various features like the Commanding Panel and Image Panel.
                    </p>
                    <Link href={prefixRoute('how-to-use')} color="primary">
                        Go to How to Use the Yamcs Plugin
                    </Link>
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Commanding Panel Setup</Card.Heading>
                <Card.Description>
                    <p>
                        Learn how to set up and configure the Commanding Panel to control and manage commands from your Yamcs server.
                    </p>
                    <Link href={prefixRoute('commanding-setup')} color="primary">
                        Go to Commanding Panel Setup
                    </Link>
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Image Panel Setup</Card.Heading>
                <Card.Description>
                    <p>
                        Learn how to set up the Static and Telemetric Image Panels to display images based on telemetry or manual input.
                    </p>
                    <Link href={prefixRoute('image-panel-setup')} color="primary">
                        Go to Image Panel Setup
                    </Link>
                </Card.Description>
            </Card>

        </PluginPage>
    );
}

export default Overview;
