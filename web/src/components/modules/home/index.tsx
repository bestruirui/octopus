import { Activity } from './activity';
import { Total } from './total';
import { StatsChart } from './chart';

export function Home() {
    return (
        <div className="space-y-6">
            <Total />
            <Activity />
            <StatsChart />
        </div>
    );
}
