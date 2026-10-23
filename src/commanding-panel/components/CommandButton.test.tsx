import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { CommandButton } from './CommandButton';

jest.mock('@grafana/runtime', () => ({
    getTemplateSrv: () => ({ replace: (value: string) => value }),
}));

describe('CommandButton accessible names', () => {
    const commandInfo = { command: { name: '/Rover/motor/drive_straight' }, endpoint: 'workshop_realtime' };

    it.each([
        { label: '', tooltip: 'Forward', name: 'Forward' },
        { label: 'Drive', tooltip: 'Forward', name: 'Drive' },
        { label: '', tooltip: '', name: commandInfo.command.name },
    ])('names a button as $name without changing its command', ({ label, tooltip, name }) => {
        const onSubmit = jest.fn();
        render(
            <CommandButton
                commandInfo={commandInfo}
                index={0}
                commandState={{ label, tooltip, icon: 'arrow-up' }}
                scopedVars={{}}
                loading={false}
                dualButtonStates={{}}
                onSubmit={onSubmit}
            />
        );
        const button = screen.getByRole('button', { name });
        expect(button).toHaveTextContent(label);
        fireEvent.click(button);
        expect(onSubmit).toHaveBeenCalledWith(commandInfo, 0);
    });
});
