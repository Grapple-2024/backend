'use client';

import './globals.css';

import { useRouter } from "next/navigation";
import { Container, Row } from "react-bootstrap";
import Information from "./components/Information";
import Footer from "./components/Footer";
import About from "./components/About";
import Pricing from './components/Pricing';
import Subscribe from './components/Subscribe';
import GetStarted from './components/GetStarted';
import LegacyNavigation from '@/components/LegacyNavigation';
import { useSignOut } from '@/hook/auth';

const Home = () => {
  const router = useRouter();

  const mutation = useSignOut();
  
  return (
    <>
      <Row style={{ margin: 0, padding: 0 }}>
        <LegacyNavigation 
          onSignOut={() => {
            mutation.mutate();
            router.refresh();
          }} 
        />
        <GetStarted />
        <Container style={{ backgroundColor: 'black' }}>
          <About />
          <Information />
          <Pricing />
          <Subscribe />
          <Footer />
        </Container>
      </Row>
    </>
  )
};

export default Home;
