import { useMobileContext } from "@/context/mobile";
import { useConfirmCode, useResendCode } from "@/hook/auth";
import { colors } from "@/util/colors";
import { useSearchParams } from "next/navigation";

import { useState } from "react";
import { Button, Container, Form, InputGroup } from "react-bootstrap";

const initialFormState = "";

const Confirm = ({
  profileType = 'coach'
}: { profileType: string }) => {
  const [code, setCode] = useState(initialFormState);
  const confirmSignUp = useConfirmCode();
  const resendSignUpCode = useResendCode();
  const searchParams = useSearchParams();
  const email = searchParams.get('email');
  const gymId = searchParams.get('gymId');

  const { isMobile } = useMobileContext();

  return (
    <Container style={{
      padding: isMobile ? 20 : '40px 100px 50px 100px',
      border: '1px solid rgba(0, 0, 0, 0.1)',
      boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
    }}>
      <Form>
        <h2 style={{ paddingBottom: 40 }}>Sign in to your account</h2>
        <InputGroup className="mb-3">
          {isMobile ? null : <InputGroup.Text>Email</InputGroup.Text>}
          <Form.Control
            placeholder={email || "Email"}
            aria-label="Code"
            aria-describedby="code"
            value={email  || ""} 
            disabled
            onChange={e => setCode(e.target.value)}
          />
        </InputGroup>
        <InputGroup className="mb-3">
          {isMobile ? null : <InputGroup.Text>Code</InputGroup.Text>}
          <Form.Control
            placeholder="Code"
            aria-label="Code"
            aria-describedby="code"
            value={code}
            onChange={e => setCode(e.target.value)}
          />
        </InputGroup>
        <Button 
          variant='dark'
          style={{
            width: '100%',
            backgroundColor: 'white',
            color: colors.black,
            marginBottom: 20
          }}
          onClick={async (e) => {
            resendSignUpCode.mutate({
              email: email || "",
            });
          }}
        >
          Re-Send Code
        </Button>
        <Button 
          variant="dark" 
          type="submit" 
          style={{ 
            width: '100%',
          }}
          onClick={async (e) => {
            e.preventDefault();
            await confirmSignUp.mutateAsync({
              email: email || "",
              code,
              ...(gymId ? { gymId } : {}),
            });
          }}  
        >
          Verify
        </Button>
      </Form>
    </Container>
  );
};

export default Confirm;