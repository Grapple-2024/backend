import styles from './BeltBadge.module.css';

// Adult BJJ belt colors
const ADULT_COLORS: Record<string, string> = {
  white:     '#fff',
  blue:      '#1a5bb5',
  purple:    '#7b2fa0',
  brown:     '#7c4a1e',
  black:     '#111',
  coral:     '#ff6b47',
  'red/white': 'split:red:#dc2626:#fff',
  red:       '#dc2626',
};

// Kids belt colors — split belts encoded as "split:color1:color2"
const KIDS_COLORS: Record<string, string> = {
  white:          '#fff',
  'grey/white':   'split:#888:#fff',
  grey:           '#888',
  'grey/black':   'split:#888:#111',
  'yellow/white': 'split:#d4a800:#fff',
  yellow:         '#d4a800',
  'yellow/black': 'split:#d4a800:#111',
  'orange/white': 'split:#e06b00:#fff',
  orange:         '#e06b00',
  'orange/black': 'split:#e06b00:#111',
  'green/white':  'split:#2d7a2d:#fff',
  green:          '#2d7a2d',
  'green/black':  'split:#2d7a2d:#111',
};

interface Props {
  system: 'adult' | 'kids';
  belt: string;
  stripes: number;
  showLabel?: boolean;
}

export default function BeltBadge({ system, belt, stripes, showLabel = true }: Props) {
  const colorMap = system === 'adult' ? ADULT_COLORS : KIDS_COLORS;
  const color = colorMap[belt] ?? '#888';
  const isSplit = color.startsWith('split:');
  const isWhite = belt === 'white';

  const renderStrip = () => {
    if (isSplit) {
      const [, c1, c2] = color.split(':');
      return (
        <span className={styles.strip}>
          <span className={styles.stripHalf} style={{ background: c1 }} />
          <span className={styles.stripHalf} style={{ background: c2 }} />
        </span>
      );
    }
    return (
      <span
        className={`${styles.strip} ${isWhite ? styles.whiteStrip : ''}`}
        style={{ width: 28 }}
      >
        <span className={styles.stripFull} style={{ background: color }} />
      </span>
    );
  };

  // Stripe pips sit on top of the belt strip color
  // For white belt: dark pips; for others: white pips
  const pipClass = isWhite ? `${styles.pip} ${styles.pipDark}` : styles.pip;

  return (
    <span className={styles.belt}>
      <span style={{ position: 'relative', display: 'inline-flex', alignItems: 'center' }}>
        {renderStrip()}
        {stripes > 0 && (
          <span className={styles.stripes} style={{ position: 'absolute', right: 2 }}>
            {Array.from({ length: stripes }).map((_, i) => (
              <span key={i} className={pipClass} />
            ))}
          </span>
        )}
      </span>
      {showLabel && <span className={styles.label}>{belt} {stripes > 0 ? `(${stripes})` : ''}</span>}
    </span>
  );
}
