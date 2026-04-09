import { useMobileContext } from "@/context/mobile";
import Image from "next/image";
import { Button, Col, Row } from "react-bootstrap";
import styles from './style.module.css';
import { useRouter } from "next/navigation";
import GrappleButton, { ButtonVariants } from "@/components/GrappleButton";

const GetStarted = () => {
  const { isMobile } = useMobileContext();
  const router = useRouter();

  return (
    <Row style={{ margin: 0, padding: 0 }}>
      {isMobile ? (
        <Col xs={12} className={styles['mobile-container']}>
          <Image
            src="/hands.png"
            fill
            style={{ objectFit: 'cover' }}
            alt="Picture of a grappler's hands"
            className={styles['header-mobile']}
          />
          <div className={styles['content-mobile']}>
            <h1 className={styles['heading']}>
              The more you you know,
            </h1>
            <h1 className={styles['subheading']}>
              the more you flow.
            </h1>
            <h6 className={styles['description']}>
              Grapple is a universal platform for those who are engulfed in the Mixed Martial Arts lifestyle. We have created The First Ever Platform that allows for coaches to connect with their students outside of the gym, and a place where students can review and learn techniques post training.
            </h6>
          </div>
            <GrappleButton
              variant={ButtonVariants.TERTIARY} 
              onClick={() => {
                router.push('/auth');
              }}
            >
            Get Started
          </GrappleButton>
        </Col>
      ) : (
        <>
          <Col xs={6} className={styles['header-desktop']}>
            <Image
              src="/hands.png"
              fill
              style={{ objectFit: 'cover' }}
              alt="Picture of a grappler's hands"
            />
          </Col>
          <Col xs={6} className={styles['col-content']}>
            <h1 className={styles['heading']}>
              The more you you know,
            </h1>
            <h1 className={styles['subheading']}>
              the more you flow.
            </h1>
            <h6 className={styles['description']}>
              Grapple is a universal platform for those who are engulfed in the Mixed Martial Arts lifestyle. We have created The First Ever Platform that allows for coaches to connect with their students outside of the gym, and a place where students can review and learn techniques post training.
            </h6>
            <GrappleButton
              variant={ButtonVariants.TERTIARY} 
              onClick={() => {
                router.push('/auth');
              }}
            >
            Get Started
          </GrappleButton>
          </Col>
        </>
      )}
    </Row>
  );
};

export default GetStarted;
