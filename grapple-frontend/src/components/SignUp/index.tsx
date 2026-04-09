import { colors } from "@/util/colors";

import { useEffect, useState } from "react";
import { Button, Container, Form, InputGroup } from "react-bootstrap";
import { FaEye, FaEyeSlash, FaInfoCircle } from "react-icons/fa";


import { useRouter } from "next/navigation";
import ToolTip from "../ToolTip";
import { useMessagingContext } from "@/context/message";
import { useMobileContext } from "@/context/mobile";
import { useSignup } from "@/hook/auth";
import { useAuth } from "@clerk/nextjs";



const initialFormState = { 
  firstName: "",
  lastName: "",
  email: "", 
  phone: "",
  password: "", 
  confirmPassword: "",
  terms: true,
};

const passwordRules = 'Password must contain at least 8 characters, one uppercase letter, one number, and one special character';
const phoneNumberRules = 'Phone number must be 11 digits long and start with 1';

const SignUp = ({
  profileType = 'coach',
  gymId = null,
}: any) => {
  const router = useRouter();

  const [formData, setFormData] = useState(initialFormState);
  const [showPassword, setShowPassword] = useState(false);
  const [emailError, setEmailError] = useState(true);
  const [phoneError, setPhoneError] = useState(true);
  const [firstNameError, setFirstNameError] = useState(true);
  const [lastNameError, setLastNameError] = useState(true);
  const [passwordError, setPasswordError] = useState(true);
  const [confirmPasswordError, setConfirmPasswordError] = useState(true);
  const { isMobile } = useMobileContext();
  const signUp = useSignup();
  const { isSignedIn } = useAuth();

  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();

  useEffect(() => {
    const isValid = validateEmail(formData.email);

    setEmailError(!isValid);
    setFirstNameError(formData.firstName === '');
    setLastNameError(formData.lastName === '');

    const isPhoneValid = validatePhoneNumber(formData.phone);

    setPhoneError(!isPhoneValid);
    
    const isPasswordValid = validatePassword(formData.password);

    setPasswordError(!isPasswordValid);
    setConfirmPasswordError(!(formData.password === formData.confirmPassword));
  }, [formData]);
  
  function validatePhoneNumber(input: string) {
    // Define a regex pattern for the updated phone number validation
    // This pattern matches strings like "19495143970" - an 11 digit number starting with '1'
    const phoneNumberPattern = /^1\d{10}$/;
    
    // Check if the input matches our phone number pattern
    if (phoneNumberPattern.test(input)) {
      // If it matches, the format is correct
      return true;
    } else {
      // If it does not match, the format is incorrect
      return false;
    }
  }

  function validateEmail(email: string) {
    // Define a regex pattern for email validation
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

    // Test the email against the pattern
    if (emailPattern.test(email)) {
      return true; // The email address is valid
    } else {
      return false;
    }
  }

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

  async function signUpUser() {
    try {
      if (!formData.email || !formData.password || !formData.confirmPassword) {
        throw new Error('Please fill out all fields');
      }

      if (formData.password !== formData.confirmPassword) {
        throw new Error('Passwords do not match');
      }

      await signUp.mutateAsync({
        username: formData.email,
        password: formData.password,
        lastName: formData.lastName,
        firstName: formData.firstName,
        phone: "+" + formData.phone,
        gymId,
      });
      // navigation is handled by the hook's onSuccess
    } catch (error: any) {
      setMessage(error?.message || 'Sign up failed. Please try again.');
      setColor('danger');
      setShow(true);
    }
  }

  return (
    <Container style={{
      padding: isMobile ? 20 : '40px 100px 50px 100px',
      border: '1px solid rgba(0, 0, 0, 0.1)',
      boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
    }}>
      <Form>
        <h2 style={{ paddingBottom: 40 }}>Create a new account</h2>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          {isMobile ? null : <InputGroup.Text>First and last name</InputGroup.Text>}
          <Form.Control 
            aria-label="First name" 
            placeholder="First Name" 
            value={formData.firstName}
            onChange={e => {
              setFormData({ ...formData, 'firstName': e.target.value });

              if (formData.firstName === '') {
                setFirstNameError(true);
              } else {
                setFirstNameError(false);
              }
            }}
            style={{
              borderColor: firstNameError ? colors.primary : '',
            }}
          />
          <Form.Control 
            aria-label="Last name" 
            placeholder="Last Name"
            value={formData.lastName}
            style={{
              borderColor: lastNameError ? colors.primary : '',
            }}
            onChange={e => {
              setFormData({ ...formData, 'lastName': e.target.value });

              if (formData.lastName === '') {
                setLastNameError(true);
              } else {
                setLastNameError(false);
              }
            }}
          />
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          { isMobile ? null : <InputGroup.Text>Email</InputGroup.Text>}
          <Form.Control
            placeholder="Email"
            aria-label="Email"
            aria-describedby="email"
            value={formData.email}
            style={{
              borderColor: emailError ? colors.primary : '',
            }}
            onChange={e => {
              setFormData({ ...formData, 'email': e.target.value });
              const isValid = validateEmail(formData.email);

              if (isValid) {
                setEmailError(false);
              } else {
                setEmailError(true);
              }
            }}
          />
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          {isMobile ? null : <InputGroup.Text>Phone</InputGroup.Text>}
          <Form.Control
            placeholder="19281115555"
            aria-label="Phone Number"
            aria-describedby="phone-number"
            value={formData.phone}
            style={{
              borderColor: phoneError ? colors.primary : '',
            }}
            onChange={e => {
              setFormData({ ...formData, 'phone': e.target.value });
              const isValid = validatePhoneNumber(formData.phone);

              if (isValid) {
                setPhoneError(false);
              } else {
                setPhoneError(true);
              }
            }}
            onBlur={() => { 
              const isValid = validatePhoneNumber(formData.phone);

              setPhoneError(!isValid);
            }}
          />
          <InputGroup.Text>
            <ToolTip text={phoneNumberRules}>
              <FaInfoCircle />
            </ToolTip>
          </InputGroup.Text>
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          {isMobile ? null : <InputGroup.Text>Password</InputGroup.Text>}
          <Form.Control 
            type={showPassword ? 'text' : 'password'}
            aria-label="password"
            placeholder="Password" 
            value={formData.password}
            style={{
              borderColor: passwordError ? colors.primary : '',
            }}
            onChange={e => {
              setFormData({ ...formData, 'password': e.target.value });
              const isValid = validatePassword(formData.password);

              setPasswordError(!isValid);
            }}
            onBlur={() => {
              const isValid = validatePassword(formData.password);

              setPasswordError(!isValid);
            }}
          />
          <InputGroup.Text style={{ backgroundColor: colors.black }}>
            {showPassword ? 
              <FaEye color={colors.white} onClick={() => {
                setShowPassword(false);
              }}/> : 
              <FaEyeSlash color={colors.white} onClick={() => {
                setShowPassword(true);
              }}/>
            }
          </InputGroup.Text>
          <InputGroup.Text>
            <ToolTip text={passwordRules}>
              <FaInfoCircle />
            </ToolTip>
          </InputGroup.Text>
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 20 }}>
          {isMobile ? null : <InputGroup.Text>Confirm Password</InputGroup.Text>}
          <Form.Control 
            type={showPassword ? 'text' : 'password'}
            aria-label="confirm-password"
            placeholder="Confirm Password" 
            style={{
              borderColor: confirmPasswordError ? colors.primary : '',
            }}
            value={formData.confirmPassword}
            onChange={e => {
              setFormData({ ...formData, 'confirmPassword': e.target.value });
              
              if (formData.password === formData.confirmPassword) {
                setConfirmPasswordError(false);
              } else {
                setConfirmPasswordError(true);
              }
            }}
            onBlur={() => {
              if (formData.password === formData.confirmPassword) {
                setConfirmPasswordError(false);
              } else {
                setConfirmPasswordError(true);
              }
            }}
          />
          <InputGroup.Text style={{ backgroundColor: colors.black }}>
            {showPassword ? 
              <FaEye color={colors.white} onClick={() => {
                setShowPassword(false);
              }}/> : 
              <FaEyeSlash color={colors.white} onClick={() => {
                setShowPassword(true);
              }}/>
            }
          </InputGroup.Text>
          <InputGroup.Text>
            <ToolTip text={passwordRules}>
              <FaInfoCircle />
            </ToolTip>
          </InputGroup.Text>
        </InputGroup>
        <Button 
          variant="dark" 
          type="submit" 
          style={{ 
            width: '100%',
          }}
          onClick={(e) => {
            e.preventDefault();
            signUpUser();
          }}
        >
          Submit
        </Button>
        <a style={{ 
            display: 'flex',
            justifyContent: 'center',
            marginTop: 30 ,
            cursor: 'pointer',
          }}
          onClick={() => {
            router.push(isSignedIn ? '/auth/account-type' : '/');
          }}
        >
          Back to home
        </a>
      </Form>
    </Container>
  );
};

export default SignUp;