import { PluginPage } from '@grafana/runtime';
import { Card, Text } from '@grafana/ui';
import React from 'react';

function TimeSyncSetup() {
    return (
        <PluginPage>
            <p>
                Welcome! This guide explains how to set up the JAOPS Yamcs Time Sync panel for replay and simulation
                workflows. It also explains what each panel option does, so you can tune stability and responsiveness.
            </p>

            <Card>
                <Card.Heading>Step 1: Add a Time Sync Panel</Card.Heading>
                <Card.Description>
                    In your dashboard, click <Text color="info">Add panel</Text>, set the data source to{' '}
                    <Text color="primary">Yamcs Datasource</Text>, and choose query type{' '}
                    <Text color="primary">Time</Text>. Then select visualization{' '}
                    <Text color="primary">JAOPS Yamcs Time Sync</Text>.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 2: Keep Dashboard Range Relative</Card.Heading>
                <Card.Description>
                    Use a relative dashboard range, for example <Text variant="code">now-15m</Text> to{' '}
                    <Text variant="code">now</Text>. Time sync works by rewriting these expressions with a Yamcs-based
                    offset during replay/simulation.
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
                            <Text color="primary">Normalize-to-now threshold (ms):</Text> If skew is under this
                            threshold, sync snaps back to plain <Text variant="code">now</Text>.
                        </li>
                    </ul>
                </Card.Description>
            </Card>
        </PluginPage>
    );
}

export default TimeSyncSetup;
