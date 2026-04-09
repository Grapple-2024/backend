import { useMessagingContext } from "@/context/message";
import { useChangePassword } from "@/hook/auth";
import { colors } from "@/util/colors";
import { useState } from "react";
import { Button, Col, Form, Row } from "react-bootstrap";
import { FaEdit } from "react-icons/fa";

interface ChangePasswordProps {
  user: any;
};

const ChangePassword = ({ user }: ChangePasswordProps) => {  
  const [isEditing, setIsEditing] = useState(false);
  const [password, setPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmNewPassword, setConfirmNewPassword] = useState('');
  const changePassword = useChangePassword();
  const {
    setMessage,
    setShow,
    setColor
  } = useMessagingContext();

  function validatePassword(password: string) {
    // Check for at least one uppercase letter
    const hasUpperCase = /[A-Z]/.test(password);
    
    // Check for at least one special character
    const hasSpecialCharacter = /[!@#$%^&*(),.?":{}|<>]/.test(password);
    
    // Check for at least one number
    const hasNumber = /\d/.test(password);
    
    // Optional: Check for minimum length, e.g., 8 characters
    const isMinLength = password.length >= 8;

    if (hasUpperCase && hasSpecialCharacter && hasNumber && isMinLength) {
        return true; // Password is valid
    } else {
        return false; // Password is invalid
    }
  }
  
  const onSubmit = async () => {
    if (!password || !newPassword || !confirmNewPassword) {
      setColor('danger');
      setMessage('Password update failed');
      setShow(true);
      setPassword('');
      setNewPassword('');
      setConfirmNewPassword('');
      return;
    }

    if (newPassword !== confirmNewPassword) {
      setColor('danger');
      setMessage('Password update failed');
      setShow(true);
      setPassword('');
      setNewPassword('');
      setConfirmNewPassword('');
      return;
    }

    if (!validatePassword(newPassword)) {
      setColor('danger');
      setMessage('Password does not meet requirements');
      setShow(true);
      setPassword('');
      setNewPassword('');
      setConfirmNewPassword('');
      return;
    }

    changePassword.mutate({
      oldPassword: password,
      newPassword: newPassword,
    });

    setPassword('');
    setNewPassword('');
    setConfirmNewPassword('');
  };

  return (
    <div style={{
      padding: 20,
      borderRadius: 10, 
      backgroundColor: 'white',
    }}>
      <Row style={{ marginBottom: 20 }}>
        <Col style={{ display: 'flex', justifyContent: 'flex-start' }}>
          <h4>Change Password</h4>
        </Col>
        <Col style={{ display: 'flex', justifyContent: 'flex-end' }}>
          {
            !isEditing ? (
              <FaEdit 
                size={20}
                onClick={() => {
                  setIsEditing(!isEditing);
                }}
              />
            ) : null
          }
        </Col>
      </Row>
      <Row>
        <Col>
          <Form.Label htmlFor="currentPasswordSettings">Current Password</Form.Label>
          <Form.Control
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
            }}
            type="password"
            id="currentPasswordSettings"
            aria-describedby="currentPasswordSettings"
            disabled={!isEditing}
          />
        </Col>
      </Row>
      <Row style={{ marginTop: 10 }}>
        <Col>
          <Form.Label htmlFor="newPasswordSettings">New Password</Form.Label>
          <Form.Control
            value={newPassword}
            onChange={(e) => {
              setNewPassword(e.target.value);
            }}
            type="password"
            id="newPasswordSettings"
            aria-describedby="newPasswordSettings"
            disabled={!isEditing}
          />
        </Col>
      </Row>
      <Row style={{ marginTop: 10 }}>
        <Col>
          <Form.Label htmlFor="confirmNewPasswordSettings">Confirm New Password</Form.Label>
          <Form.Control
            value={confirmNewPassword}
            onChange={(e) => {
              setConfirmNewPassword(e.target.value);
            }}
            type="password"
            id="confirmNewPasswordSettings"
            aria-describedby="confirmNewPasswordSettings"
            disabled={!isEditing}
          />
        </Col>
      </Row>
      {
        isEditing ? (
          <Row>
            <Col style={{
              display: 'flex',
              justifyContent: 'flex-end',
            }}>
              <Button 
                variant="dark"
                onClick={() => setIsEditing(!isEditing)}
                style={{ 
                  marginTop: 20, 
                  marginRight: 10,
                  backgroundColor: 'white', 
                  borderColor: 'black', 
                  color: 'black',
                }} 
              >
                Cancel
              </Button>
              <Button style={{ 
                  marginTop: 20, 
                  backgroundColor: colors.black, 
                  borderColor: colors.black, 
                }} 
                type="submit"
                onClick={async () => {
                  onSubmit();
                  setIsEditing(!isEditing);
                }}
              >
                Update Password
              </Button>
            </Col>
          </Row>
        ) : null
      }
    </div>
  );
};

export default ChangePassword;