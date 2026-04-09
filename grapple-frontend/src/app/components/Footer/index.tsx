import { colors } from "@/util/colors";
import { Col, Row } from "react-bootstrap";
import { FaFacebookF, FaInstagram, FaTiktok } from "react-icons/fa";


const Footer = () => {
  return (
    <Row style={{ 
      height: 100,
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center', 
    }}>
      <Col style={{ textAlign: 'end', color: colors.white }} xs={6}>
        {/* <h5>
          Our socials:
        </h5> */}
        <h5>
          Have an issue? Contact us at 1.949.226.2229
        </h5>
      </Col>
      {/* <Col xs={6}>
        <FaFacebookF size={25} color={colors.white}/>
        <FaInstagram size={25} color={colors.white} style={{ marginLeft: 25 }} />
        <FaTiktok size={25} color={colors.white} style={{ marginLeft: 25 }} />
      </Col> */}
    </Row>
  );
};

export default Footer;