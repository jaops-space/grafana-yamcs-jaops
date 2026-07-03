import { PluginPage } from '@grafana/runtime';
import { Alert, Badge, Card, Text } from '@grafana/ui';
import React from 'react';

function TimeSyncSetup() {
    return (
        <PluginPage>
            <p>
                Welcome! This guide explains how to set up the JAOPS Yamcs Time Sync panel for replay and simulation
                workflows. It also explains what each panel option does, so you can tune stability and responsiveness.
            </p>

            <Card>
                <Card.Heading>Step 1: Setup a Replay processor on Yamcs</Card.Heading>
                <Card.Description>
                    The first step is to set up a replay processor, if you already have it set up, you might skip this step.
                    <br/><br/>
                    The provisioned datasource already has a setup for a replay processor called <code>replay</code>, all you need to do is start a replay on <b>Yamcs web</b> with that same name, and change the <code>endpoint</code> variable to <code>myproject_replay</code> or <code>simulator_replay</code> on the Demo dashboard.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 2: Add a Time Sync Panel</Card.Heading>
                <Card.Description>
                    In your dashboard, click <Text color="info">Add panel</Text>, set the data source to{' '}
                    <Text color="primary">Yamcs Datasource</Text>, and choose query type{' '}
                    <Text color="primary">Time</Text>. Then select visualization{' '}
                    <Text color="primary">JAOPS Yamcs Time Sync</Text>.
                    <br />
                    <br />
                    Make sure to choose the right endpoint for your time sync panel, if you use multiple endpoints
                    through a variable, make sure to use that variable as the endpoint.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 3: Recommended Replay/Simulation Setup</Card.Heading>
                <Card.Description>
                    <ul>
                        <li>
                            Enable <Text color="primary">Enable Yamcs time sync</Text>.
                        </li>
                        <li>
                            Keep <Text color="primary">Only apply when range is relative</Text> enabled.
                        </li>
                        <li>
                            Start with <Text variant="code">Offset step (ms) = 15000</Text> and{' '}
                            <Text variant="code">Minimum write interval (ms) = 10000</Text> for stable updates.
                        </li>
                    </ul>
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 4: Keep Dashboard Range Relative</Card.Heading>
                <Card.Description>
                    Use a relative dashboard range, for example <Text variant="code">now-15m</Text> to{' '}
                    <Text variant="code">now</Text>. Time sync works{' '}
                    <Text color="primary">by rewriting these expressions with an offset during replay/simulation</Text>.
                    <br />
                    <br />
                    E.g. if browser time is <code>30m</code> ahead of Yamcs clock, the panel rewrites the range to{' '}
                    <code>now-45m</code> to <code>now-30m</code>.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Panel Option Reference</Card.Heading>
                <Card.Description>
                    <ul>
                        <li>
                            <Text color="primary">Enable Yamcs time sync:</Text> Master on/off switch. When off, the
                            panel does not rewrite dashboard time.
                        </li>
                        <li>
                            <Text color="primary">Show status card:</Text> Shows a compact status state in the panel
                            (functional / not functional / disabled).
                        </li>
                        <li>
                            <Text color="primary">Only apply when range is relative (contains now):</Text> Safety option
                            to avoid changing absolute time ranges.
                        </li>
                        <li>
                            <Text color="primary">Offset step (ms):</Text> The step size used to round the computed
                            offset (for example, 15,000 ms means offsets move in 15-second steps). Larger steps make
                            updates less jittery.
                        </li>
                        <li>
                            <Text color="primary">Minimum write interval (ms):</Text> How often sync is allowed to
                            update the dashboard time. Larger values mean fewer updates.
                        </li>
                        <li>
                            <Text color="primary">Normalize-to-now threshold (ms):</Text> If skew (offset between local
                            time and Yamcs clock) is under this threshold, sync snaps back to plain{' '}
                            <Text variant="code">now</Text> and shows <Badge text={'REALTIME'} color="blue" />.
                        </li>
                    </ul>
                </Card.Description>
            </Card>
            <Alert title="Important to know" severity="warning">
                An important caveat to know is when Yamcs replay speed is different than original speed (<code>1x</code>
                ).
                <br />
                Time synchronization works with sped up replays, but will cause the sync to rewrite the time range to
                catch up, and panels to refresh every few seconds, specifically every time the skew surpasses{' '}
                <b>Offset step</b>.
                <br />
                In between refreshes, the panel will only move at normal speed. The visual intuition is to follow the
                movement of refreshes, which will catch up with the Yamcs clock. However, that comes with a potential
                performance issue (frequent refreshes are expensive).
            </Alert>
        </PluginPage>
    );
}

export default TimeSyncSetup;
