import { Col, Container, Row } from "react-bootstrap";
import PricingCard from "../PricingCard";
import { useMobileContext } from "@/context/mobile";
import styles from './style.module.css';

const planContents = [
  {
    header: "Basic",
    price: 55,
    features: [
      "Full access to building a secure media library for your gym",
      "Fully control who can access your gyms content",
      "Create and upload video series to show the progression of a drill or technique",
      "Allow assistant coaches administrative access to upload content and announcements for their classes",
      "Organize your gym's videos by coach, difficulty, and discipline",
      "Send bulk announcements to your students directly from the dashboard",
      "Upload your gym's schedule directly to the dashboard",
      "Pin videos each week to help your students stay on track with the curriculum",
      "Customize your public profile for your gym to expand your business reach",
    ],
    buttonLabel: "Get Started",
    outline: true,
    center: false,
    src: '/grapple-price-1.png',
    path: '/coach/auth',
    position: 'left'
  },
  {
    header: "Pro",
    price: 100,
    features: [
      'All features from "Basic" as well as',
      "Full membership management",
      "Sign up and liability form templates",
      "View business charts and statistics",
      "Sell video series publicly as a coach from your public profile",
      "Sell your virtual gym memberships for your gym from your public profile",
      "Give your students full access to manage their memberships via you gym dashboard",
      "Tax preparation forms to help manage additional streams of income"
    ],
    buttonLabel: "Get Started",
    outline: false,
    center: true,
    src: '/grapple-price-1.png',
    position: 'middle'
  },
  {
    header: "Enterprise",
    price: '~',
    features: [
      "Own multiple gyms? Contact sales for a custom plan",
    ],
    buttonLabel: "Contact Us",
    outline: false,
    center: false,
    src: '/grapple-price-1.png',
    position: 'right'
  }
];

const Pricing = () => {
  const { isMobile } = useMobileContext();

  const plans = planContents.map((obj, i) => {
    return (
      <Col xs={isMobile ? 12 : 4} key={i}>
        <PricingCard
          key={obj.header}
          header={obj.header}
          price={obj.price as any}
          features={obj.features}
          buttonLabel={obj.buttonLabel}
          outline={obj.outline}
          center={obj.center}
          src={obj.src}
          position={obj.position}
          path={obj.path}
        />
      </Col>
    );
  });

  return (
    <Container className={styles.container}>
      <Row>
        <h2 className={styles.heading}>
          Pricing
        </h2>
      </Row>
      <Row>
        {plans}
      </Row>
    </Container>
  );
};

export default Pricing;
