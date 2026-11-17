import { aliasesToText, textToAliases } from '../config/ConfigHost';

describe('host aliases', () => {
    it('reads one instance=ID per line, ignoring blank and partial lines', () => {
        expect(textToAliases('fm1=sat-1\n\n fm2 = sat-2 \nbroken\n=x')).toEqual({ fm1: 'sat-1', fm2: 'sat-2' });
    });

    it('writes them back', () => {
        expect(aliasesToText({ fm1: 'sat-1', fm2: 'sat-2' })).toBe('fm1=sat-1\nfm2=sat-2');
        expect(aliasesToText(undefined)).toBe('');
    });
});
