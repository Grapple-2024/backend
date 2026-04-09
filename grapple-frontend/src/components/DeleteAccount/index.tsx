import { colors } from "@/util/colors";
import { Button, Col, Container, Row } from "react-bootstrap";
import DeleteAccountModal from "./components/DeleteAccountModal";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useMobileContext } from "@/context/mobile";


const DeleteAccount = () => {
  const [show, setShow] = useState(false);
  const router = useRouter();  

  return (
    <div style={{
      padding: 20,
      backgroundColor: 'white',
      borderRadius: 10, 
    }}>
      <Row>
        <Col style={{
          display: 'flex',
          justifyContent: 'flex-start',
          alignItems: 'center',
        }}>
          <h4 className='text-center'>Delete Account</h4>
        </Col>
        <Col style={{
          display: 'flex',
          justifyContent: 'flex-end',
        }}>
          <Button style={{ 
            backgroundColor: colors.primary, 
            borderColor: colors.primary,
          }} onClick={() => {setShow(true)}}>
            Delete Account
          </Button>
        </Col>
      </Row>
      <DeleteAccountModal 
        show={show} 
        onHide={() => setShow(false)}
        onDeleted={async () => {
          try {
            // await deleteUser();
            router.push('/auth');
          } catch (error) {
            console.error('Error deleting account', error);
          }
        }}
      />
    </div>
  );
};

export default DeleteAccount;