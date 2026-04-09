import { useEffect, useRef, useState } from "react";
import { Button, Card } from "react-bootstrap";
import styles from './style.module.css';
import { useRouter } from "next/navigation";

interface Props {
  header: string;
  price: number;
  features: string[];
  buttonLabel: string;
  outline?: boolean;
  center?: boolean;
  src: string;
  position?: string;
  path?: string;
}

const PricingCard = ({
  header,
  price,
  features,
  buttonLabel,
  outline,
  center,
  src,
  position,
  path,
}: Props) => {
  const cardRef = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);
  const router = useRouter();

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.disconnect(); // Stop observing once it has become visible
        }
      },
      {
        threshold: 0.1,
      }
    );

    if (cardRef.current) {
      observer.observe(cardRef.current);
    }

    return () => {
      if (cardRef.current) {
        observer.unobserve(cardRef.current);
      }
    };
  }, []);

  return (
    <Card
      ref={cardRef}
      className={`mb-1 shadow-sm ${styles.card} ${isVisible ? styles.cardVisible : ''} ${center ? styles.cardCenter : styles.cardNotCenter} ${position === 'middle' ? styles.cardMiddle : styles.cardLeftRight}`}
    >
      <Card.Header>
        <h4 className={`my-0 ${styles.cardTitle}`}>{header}</h4>
      </Card.Header>
      <Card.Img 
        variant="top" 
        src={src}
        className={styles.cardImg}
      />
      <Card.Body className={styles.cardBody}>
        <div>
          <Card.Title className="text-center mb-3">
            {`$${price}`}
            <small className="text-muted">/ mo</small>
          </Card.Title>
          <ul className="list-unstyled mt-3 mb-4 text-left">
            {features.map((feature, i) => (
              <li key={i} className="mb-2">{feature}</li>
            ))}
          </ul>
        </div>
        <Button 
          variant="dark" 
          className={`btn btn-lg ${styles.cardButton} ${outline ? styles.cardButtonOutline : ''}`}
          onClick={() => router.push(path || '')}
        >
          {buttonLabel}
        </Button>
      </Card.Body>
    </Card>
  );
}

export default PricingCard;
