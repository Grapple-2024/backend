'use client';

import SignIn from "@/components/SignIn";
import SignUp from "@/components/SignUp";
import { colors } from "@/util/colors";
import { Spinner, Tab, Tabs } from "react-bootstrap";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { useMobileContext } from "@/context/mobile";

const AuthContent = () => {
  const { isMobile } = useMobileContext();
  const searchParams = useSearchParams();
  const gymId = searchParams.get('gym_id');
  
  return (
    <div style={{
      marginLeft: isMobile ? 10 : '25%',
      marginRight: isMobile ? 10 : '25%',
      paddingTop: isMobile ? '25%' : '5%',
    }}>
      {
        gymId ? (
          <>
            <h2 style={{ color: colors.black, textAlign: 'center', marginBottom: 20 }}>
              Please create an account or sign in to join your gym
            </h2>
          </>
        ): null
      }
      <Tabs
        defaultActiveKey={gymId ? "register" : "sign-in"}
        id="fill-tab-example"
        fill
      >
        <Tab eventKey="sign-in" 
          title={(
            <span style={{ color: colors.black }}>Sign In</span>
          )}
        >
          <SignIn gymId={gymId}/>
        </Tab>
        <Tab eventKey="register" title={(
          <span style={{ color: colors.black }}>Create Account</span>
        )}>
          <SignUp gymId={gymId}/>
        </Tab>
      </Tabs>
    </div>
  );
};

export default AuthContent;