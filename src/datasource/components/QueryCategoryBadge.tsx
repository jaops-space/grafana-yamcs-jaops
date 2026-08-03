import { Badge } from '@grafana/ui';
import React from 'react';
import { QueryCategory } from './constants';

interface Props {
    category: QueryCategory;
}

export function QueryCategoryBadge({ category }: Props): React.ReactNode {
    switch (category) {
        case QueryCategory.PARAMETER:
            return <Badge color="blue" text="Parameter" />;
        case QueryCategory.TIMELINE:
            return <Badge color="purple" text="Timeline" />;
        case QueryCategory.IMAGE:
            return <Badge color="green" text="Image" />;
        case QueryCategory.COMMANDING:
            return <Badge color="red" text="Commanding" />;
        case QueryCategory.ALARMS:
            return <Badge color="orange" text="Alarm" />;
        case QueryCategory.LINKS:
            return <Badge color="blue" text="Links" />;
        case QueryCategory.DEBUG:
            return <Badge color="orange" text="Debug" />;
        default:
            return <Badge color="red" text="Unknown" />;
    }
}
