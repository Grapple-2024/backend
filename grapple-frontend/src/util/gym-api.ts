import axios from 'axios';

const gymApi = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_HOST}`,
});


export default gymApi;