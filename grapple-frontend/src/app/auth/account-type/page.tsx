'use client';

import { Col, Row, Spinner } from "react-bootstrap";
import { useState } from "react";

import { FaArrowRight } from "react-icons/fa";
import styles from './AccountTypePage.module.css';
import { useRouter } from "next/navigation";
import Image from "next/image";

const AccountTypePage = () => {
  const [accountType, setAccountType] = useState<string>("student");
  const router = useRouter();
  const [isLoading, setIsLoading] = useState<boolean>(false);

  return (
    <div className={styles.container}> 
      <Row>
        <h2 className={styles.title}>
          Welcome to Grapple
        </h2>
      </Row>
      <Row className={styles.content}> 
        <Col className={styles.imageCol}>
          <div className={styles.imageContainer}>
            <Image
              src="/hands.png"
              className={styles.image}
              fill
              priority
              alt="Loading header"
            />
          </div>
        </Col>
        <Col className={styles.selectionCol}>
          <div className={styles.selectionContainer}>
            {
              isLoading ? (
                <div className='text-center'>
                  <Spinner />
                </div>
              ) : (
                <>
                  <h4 className={styles.subtitle}>
                    Are you a gym owner or student?
                  </h4>
                  <div className={styles.radioContainer}>
                    <div className={styles.radioOption}>
                      <input
                        type="radio"
                        id="owner"
                        name="accountType"
                        value="owner"
                        checked={accountType === "owner"}
                        onChange={(e) => setAccountType(e.target.value)}
                      />
                      <label htmlFor="owner">Owner</label>
                    </div>
                    <div className={styles.radioOption}>
                      <input
                        type="radio"
                        id="student"
                        name="accountType"
                        value="student"
                        checked={accountType === "student"}
                        onChange={(e) => setAccountType(e.target.value)}
                      />
                      <label htmlFor="student">Student</label>
                    </div>
                  </div>
                </>
              )
            }
          </div>
          <button disabled={isLoading} className={styles.nextButton} onClick={() => {
              setIsLoading(true);
              if (accountType === "owner") {
                router.push('/coach/create-gym')
              } else {
                router.push('/student/my-gym');
              }
            }}>
            <FaArrowRight size={24} />
          </button>
        </Col>
      </Row>
    </div>
  );
};

export default AccountTypePage;