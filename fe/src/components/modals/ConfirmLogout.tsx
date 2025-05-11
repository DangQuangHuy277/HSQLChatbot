
import { motion } from "framer-motion";
import classNames from "classnames";
import { IonIcon } from "@ionic/react";
import { closeOutline } from "ionicons/icons";
import { useAuth, useSettings } from "../../store/store";

const varinats = {
  hidden: { opacity: 0, scale: 0.8 },
  visible: { opacity: 1, scale: 1 },
  exit: { opacity: 0, scale: 0.8, transition: { duration: 0.15 } },
};
export default function ConfirmLogout() {
const { logout } = useAuth();
  const setConfirmLogout = useSettings((state) => state.setConfirmLogout);

  return (
    <motion.div
      variants={varinats}
      initial="hidden"
      animate="visible"
      exit="exit"
      className="tabs font-bold rounded-md bg-white dark:bg-gray-800 mx-2 md:mx-0 text-gray-500 dark:text-gray-300 w-full max-w-xl py-4 transition-all"
    >
      <div className="relative flex items-center justify-between px-4 py-2">
        <div className="text-center justify-center items-center w-full">
        <label htmlFor="sysmsg" className=" inline-block mb-2">
            Are you sure you want to logout?
        </label>

        <div className="flex justify-center mt-2">
          <button
            className=" bg-teal-700 hover:bg-teal-900 text-white px-4 py-2 rounded mr-2"
            type="submit"
            onClick={() => {
                // Add your logout logic here
                console.log("Logged out");
                logout();
                setConfirmLogout(false);
              }}
          >
            Logout
          </button>
          <button
            className="bg-gray-200 hover:bg-gray-400 text-gray-700 px-4 py-2 rounded"
            onClick={() => setConfirmLogout(false)}
          >
            Cancel
          </button>
        </div>
        </div>
        {/* <div className="absolute top-0 right-0 items-center">
          <button
            className={classNames(" hover:text-red-300 text-2xl")}
            onClick={() => setConfirmLogout(false)}
          >
            <IonIcon icon={closeOutline} />
          </button>
        </div> */}
      </div>
    </motion.div>
  );
}
