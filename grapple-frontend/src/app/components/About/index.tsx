import { Carousel, Container, Image, Row } from "react-bootstrap";
import styles from './style.module.css';
import GrappleButton, { ButtonVariants } from "@/components/GrappleButton";
import { useRouter } from "next/navigation";

const About = () => {
  const router = useRouter();

  return (
    <Container className={styles.container}>
      <Row className={styles['text-center']}>
        <h2 className={styles.quote}>
          “Repetition is the key to mastery.” - Malcolm Gladwell
        </h2>
      </Row>
      <Row className={styles['center-content']}>
        <GrappleButton
          variant={ButtonVariants.TERTIARY} 
          onClick={() => {
            router.push('/auth');
          }}
        >
          Learn More
        </GrappleButton>
      </Row>
      <Row>
        <Carousel className={styles.carousel}>
          <Carousel.Item className={styles['carousel-item']}>
            <Image 
              src='/application-1.png'
              className={styles.image}
            />
          </Carousel.Item>
          <Carousel.Item className={styles['carousel-item']}>
            <Image 
              src='/application-2.png'
              className={styles.image}
            />
          </Carousel.Item>
          <Carousel.Item className={styles['carousel-item']}>
            <Image 
              src='/application-3.png'
              className={styles.image}
            />
          </Carousel.Item>
          <Carousel.Item className={styles['carousel-item']}>
            <Image 
              src='/application-4.png'
              className={styles.image}
            />
          </Carousel.Item>
          <Carousel.Item className={styles['carousel-item']}>
            <Image 
              src='/application-5.png'
              className={styles.image}
            />
          </Carousel.Item>
        </Carousel>
      </Row>
    </Container>
  );
};

export default About;
