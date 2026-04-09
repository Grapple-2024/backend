import { useState } from 'react';
import { Container, Row, Col, Form, Button } from 'react-bootstrap';
import styles from './style.module.css';
import { useSendEmail } from '@/hook/email';

const Subscribe = () => {
  const [email, setEmail] = useState('');
  const subscribe = useSendEmail();

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    subscribe.mutate({ email });
    setEmail('');
  };

  return (
    <Container className={styles.container}>
      <Row className="align-items-center">
        <Col md={6}>
          <h1 className={styles.header}>Don’t miss out</h1>
          <p className={styles.paragraph}>
            Show up to your next gym with confidence. Subscribe to our newsletter for the latest updates.
          </p>
        </Col>
        <Col md={6}>
          <Form onSubmit={handleSubmit} className={styles.form}>
            <Form.Group controlId="formBasicEmail" className={styles.formGroup}>
              <Form.Label className={styles.label}>Email Address</Form.Label>
              <Form.Control
                type="email"
                placeholder="Enter your email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className={styles.input}
              />
              <Form.Text className={styles.text}>
                I have read and acknowledge Grapple&apos;s <a href="#" className={styles.link}>Privacy Policy</a>
              </Form.Text>
            </Form.Group>
            <Button variant="primary" type="submit" className={styles.button}>
              Sign up
            </Button>
          </Form>
        </Col>
      </Row>
    </Container>
  );
};

export default Subscribe;
