import { IonButton, IonContent, IonHeader, IonInput, IonItem, IonLabel, IonPage, IonTitle, IonToolbar } from '@ionic/react';
import { useState } from 'react';
import { useAuth } from '../../store/store';
import axios from 'axios';

const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const { login } = useAuth();

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      const res = await axios.post('http://127.0.0.1:8080/api/v1/login', { username, password });
      login(res.data.token);
    } catch (error) {
      console.error(error);
    }
  }; 

  return (
    <IonPage className="font-montserrat">
      <IonHeader>
        <IonToolbar className="bg-[#343541]">
          <IonTitle className="text-black">Login</IonTitle>
        </IonToolbar>
      </IonHeader>
      <IonContent fullscreen className="bg-[#343541]">
        <div className="max-w-md mx-auto p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <IonItem className="bg-[#202123] rounded-md">
              <IonLabel position="floating" className="text-gray-300">Username</IonLabel>
              <IonInput
                type="text"
                value={username}
                onIonChange={(e) => setUsername(e.detail.value as string)}
                className="text-black"
              />
            </IonItem>
            <IonItem className="bg-[#202123] rounded-md">
              <IonLabel position="floating" className="text-gray-300">Password</IonLabel>
              <IonInput
                type="password"
                value={password}
                onIonChange={(e) => setPassword(e.detail.value as string)}
                className="text-black"
              />
            </IonItem>
            <IonButton type="submit" expand="block" className="mt-6 bg-[#202123] text-white">
              Login
            </IonButton>
          </form>
        </div>
      </IonContent>
    </IonPage>
  );
};

export default LoginPage;
