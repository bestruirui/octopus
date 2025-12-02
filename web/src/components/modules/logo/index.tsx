export default function Logo({ size = 48 }) {
    return (
        <svg
            viewBox="0 0 100 100"
            xmlns="http://www.w3.org/2000/svg"
            width={size}
            height={size}
            className="text-primary"
        >
            <path d="M50 15 C70 15 85 30 85 50 C85 65 75 75 70 80 M50 15 C30 15 15 30 15 50 C15 65 25 75 30 80" fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" />
            <path d="M30 80 Q30 90 20 90" fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" />
            <path d="M43 77 Q43 90 38 90" fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" />
            <path d="M57 77 Q57 90 62 90" fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" />
            <path d="M70 80 Q70 90 80 90" fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" />
        </svg>
    );
}
