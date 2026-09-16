import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import React from 'react';
import { Button, Card, FieldSet, LoadingPlaceholder, useStyles2 } from '@grafana/ui';
import { CommandEditor } from './CommandEditor';
import { VariableEditor } from './VariableEditor';
import {
    CommandErrors,
    CommandInfo,
    DualButtonStates,
    DualCommandInfos,
    UpdateArgument,
    UpdateFormOption,
    ValidateArgument,
} from '../types';

const getStyles = (theme: GrafanaTheme2) => ({
    card: css({
        width: '100%',
        padding: `${theme.spacing(1.5)} ${theme.spacing(1.75)}`,
    }),
    header: css({
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: theme.spacing(1.5),
        width: '100%',
    }),
    title: css({
        margin: 0,
    }),
    fieldSet: css({
        display: 'flex',
        flexDirection: 'column',
        gap: 0,
        width: '100%',
    }),
});

export function CommandCard(props: {
    commandInfo: CommandInfo;
    index: number;
    commandState: any;
    variableMode: boolean;
    scopedVars: any;
    loading: boolean;
    datasource: any;
    errors: CommandErrors;
    dualCommandInfos: DualCommandInfos;
    dualButtonStates: DualButtonStates;
    onSubmit: (commandInfo: CommandInfo, index: number, isOffCommand?: boolean) => void;
    onArgumentChange: UpdateArgument;
    onOptionChange: UpdateFormOption;
    onValidate: ValidateArgument;
    fetchDualCommandInfo: (commandKey: string, side: 'on' | 'off', commandName: string, endpoint: string) => void;
    clearDualCommandInfo: (commandKey: string, side: 'on' | 'off') => void;
    showPreview?: boolean;
}) {
    const {
        commandInfo,
        index,
        commandState,
        variableMode,
        scopedVars,
        loading,
        datasource,
        errors,
        dualCommandInfos,
        dualButtonStates,
        onSubmit,
        onArgumentChange,
        onOptionChange,
        onValidate,
        fetchDualCommandInfo,
        clearDualCommandInfo,
        showPreview = true,
    } = props;
    const command = commandInfo.command;
    const styles = useStyles2(getStyles);

    return (
        <Card key={`${command.name}${index}`} className={styles.card}>
            <Card.Heading>
                <div className={styles.header}>
                    <h4 className={styles.title}>{variableMode ? 'Variable Panel' : `Command Button ${index + 1}`}</h4>
                    {!variableMode && (
                        <Button disabled={loading} onClick={() => onSubmit(commandInfo, index)} size="sm">
                            {loading ? <LoadingPlaceholder text="Issuing..." /> : 'Issue Command'}
                        </Button>
                    )}
                </div>
            </Card.Heading>
            <Card.Meta>
                {variableMode ? 'Configure Grafana variables through buttons' : 'Configure a runtime command button'}
            </Card.Meta>
            <Card.Description>
                <FieldSet className={styles.fieldSet}>
                    {variableMode ? (
                        <VariableEditor
                            commandInfo={commandInfo}
                            index={index}
                            commandState={commandState}
                            scopedVars={scopedVars}
                            loading={loading}
                            dualButtonStates={dualButtonStates}
                            onOptionChange={onOptionChange}
                            showPreview={showPreview}
                        />
                    ) : (
                        <CommandEditor
                            commandInfo={commandInfo}
                            index={index}
                            commandState={commandState}
                            scopedVars={scopedVars}
                            loading={loading}
                            datasource={datasource}
                            errors={errors}
                            dualCommandInfos={dualCommandInfos}
                            dualButtonStates={dualButtonStates}
                            onArgumentChange={onArgumentChange}
                            onOptionChange={onOptionChange}
                            onValidate={onValidate}
                            fetchDualCommandInfo={fetchDualCommandInfo}
                            clearDualCommandInfo={clearDualCommandInfo}
                            showPreview={showPreview}
                        />
                    )}
                </FieldSet>
            </Card.Description>
        </Card>
    );
}
