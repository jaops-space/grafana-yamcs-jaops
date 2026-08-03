import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { PluginPage } from '@grafana/runtime';
import { Card, Alert, Text, Stack, useStyles2 } from '@grafana/ui';
import React from 'react';
import { QueryCategoryBadge } from '../datasource/components/QueryCategoryBadge';
import { QueryCategory } from '../datasource/components/constants';
import ConfigurationImage from '../img/how-to-use/configuration.png';

const getStyles = (theme: GrafanaTheme2) => ({
    queryList: css`
        display: grid;
        gap: ${theme.spacing(2)};
        margin-top: ${theme.spacing(1)};
        padding-left: ${theme.spacing(3)};
    `,
    nestedList: css`
        display: grid;
        gap: ${theme.spacing(1)};
        margin-top: ${theme.spacing(1)};
        padding-left: ${theme.spacing(4)};
    `,
    image: css`
        display: block;
        margin: ${theme.spacing(2.5)} auto;
    `,
});

function HowToUse() {
    const styles = useStyles2(getStyles);

    return (
        <PluginPage>
            <p>
                Welcome! This guide will help you set up and use the Yamcs Datasource in Grafana. Follow the steps below
                to get started.
            </p>

            <Card>
                <Card.Heading>
                    <span data-testid="jaops-setup-page-how-to-use">Step 1: Add the Datasource</span>
                </Card.Heading>
                <Card.Description>
                    Navigate to <Text color="info">Connections &gt; Data sources</Text>, then click{' '}
                    <Text color="info">Add a new data source</Text>. Search for{' '}
                    <Text color="primary">Yamcs Datasource</Text> and select it.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 2: Configure the Datasource</Card.Heading>
                <Card.Description>
                    <p>
                        The data source needs to know what to connect to, use the following configuration items for your
                        needs:
                    </p>
                    <ul>
                        <li>
                            <Text color="primary">Host:</Text> The path to your Yamcs server (e.g.,{' '}
                            <code>your-yamcs-server:8090</code>).
                        </li>
                        <li>
                            <Text color="primary">Endpoint:</Text> The instance and processor inside that host.
                        </li>
                    </ul>
                    <p>
                        Once configured, click <Text color="success">Save &amp; Test</Text> to verify the connection.
                    </p>
                </Card.Description>
            </Card>

            <img src={ConfigurationImage} width={1200} className={styles.image} />

            <Alert title="Important" severity="info">
                If you are manually running the plugin through the plugin repository Grafana docker container, to refer
                to local instances (running at <code>localhost</code>), either use <code>docker.gateway</code> or the
                proper platform-specific hostname as the <b>host path</b>.
                <br />
                <br />
                The docker container is configured to resolve <code>docker.gateway</code> to{' '}
                <code>host.docker.internal</code> for Windows, and <code>172.17.0.1</code> for Linux.
                <br />
                <br />
                If you are using the plugin in a custom Docker environment, use the proper hostname.
            </Alert>

            <Card>
                <Card.Heading>Step 3: Create a Dashboard</Card.Heading>
                <Card.Description>
                    Go to <Text color="info">Dashboards &gt; Create Dashboard</Text>, then{' '}
                    <Text color="info">Create Visualization</Text>. Select <Text color="primary">JAOPS Yamcs Datasource</Text>{' '}
                    as your data source.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 4: Start Querying</Card.Heading>
                <Card.Description>
                    <p>Choose a query type, search for a parameter and start querying data.</p>
                    <ul className={styles.queryList}>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.PARAMETER} />
                                <Text color="primary">Parameter Query</Text>
                                Fetch real-time or historical telemetry data.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Graph:</Text> Queries historical data over a time range and
                                    subscribes to live data. Used mainly for time series and plotting numerical data over
                                    time. Optional minimum and maximum fields can be added for richer graph panels.
                                </li>
                                <li>
                                    <Text color="primary">Single Value:</Text> Subscribes to the latest value without
                                    querying historical data. It can still display the current Yamcs value even when that
                                    value was generated earlier. Used mainly for stat or text-style panels.
                                </li>
                                <li>
                                    <Text color="primary">Discrete State:</Text> Queries historical ranges and subscribes
                                    to live data for discrete values such as strings, booleans, enumerations, and
                                    integers. Used mainly for state timeline panels and supports automatic colors.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.TIMELINE} />
                                <Text color="primary">Timeline Query</Text>
                                Retrieve time-based Yamcs information.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Time:</Text> Displays the current Yamcs processor time and
                                    replay speed. Used mainly with the Yamcs Time Sync panel to synchronize Grafana time
                                    with Yamcs time.
                                </li>
                                <li>
                                    <Text color="primary">Events:</Text> Lists past events over the selected time range
                                    and subscribes to live events from Yamcs.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.IMAGE} />
                                <Text color="primary">Image Query</Text>
                                Visualize images from Yamcs telemetry.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Image:</Text> Subscribes to an image parameter and displays the
                                    latest image data. Used mainly with the Telemetric Image Panel.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.COMMANDING} />
                                <Text color="primary">Commanding Query</Text>
                                Send commands and inspect command execution.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Commanding:</Text> Loads command metadata for the selected Yamcs
                                    command so the Commanding Panel can render configurable command controls.
                                </li>
                                <li>
                                    <Text color="primary">Command History:</Text> Lists command history over the selected
                                    time range and subscribes to live command history updates.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.ALARMS} />
                                <Text color="primary">Alarm Query</Text>
                                Monitor active Yamcs alarms.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Alarms:</Text> Monitors active Yamcs alarms and receives live
                                    alarm updates. Used mainly with the Alarms Panel.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.LINKS} />
                                <Text color="primary">Links Query</Text>
                                View and manage Yamcs data links.
                            </Stack>
                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Links:</Text> Lists Yamcs data links and subscribes to live link
                                    updates. Used mainly with the Links Panel to inspect, enable, and disable links.
                                </li>
                            </ul>
                        </li>
                        <li>
                            <Stack direction="row" alignItems="center" gap={1}>
                                <QueryCategoryBadge category={QueryCategory.DEBUG} />
                                <Text color="primary">Debug Query</Text>
                                These query types are available when debug mode is enabled.
                            </Stack>

                            <ul className={styles.nestedList}>
                                <li>
                                    <Text color="primary">Endpoint Stream Demands:</Text> Shows which Grafana stream paths
                                    are currently demanding Yamcs parameters from an endpoint.
                                </li>
                                <li>
                                    <Text color="primary">Yamcs Subscriptions:</Text> Shows the active Yamcs parameter
                                    subscriptions held by the endpoint.
                                </li>
                            </ul>
                        </li>
                    </ul>
                </Card.Description>
            </Card>

            <Alert title="Tip" severity="info">
                If you&apos;re facing issues, double-check your host and endpoint settings. A correct setup requires at
                least one host and one endpoint.
            </Alert>
        </PluginPage>
    );
}

export default HowToUse;
