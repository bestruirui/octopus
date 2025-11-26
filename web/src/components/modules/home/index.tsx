import { Activity } from './activity';

export function Home() {
    return (
        <div className="space-y-6">
            <div className="rounded-3xl bg-card border-card-border border p-6 text-card-foreground custom-shadow">
                <div className="h-10" />
            </div>
            <Activity />
            <div className="rounded-3xl bg-card border-card-border border  p-6 text-card-foreground custom-shadow">
                <div className="h-64" />
            </div>
        </div>
    );
}
