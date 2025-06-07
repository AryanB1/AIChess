import React, { useState, useEffect } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';
import styles from './ChessBoard.module.css';

function ChessBoard() {
    // Initializes the chess engine
    const engine = new Chess();
    // Defines the board orientation types
    type BoardOrientation = 'white' | 'black';
    // State to manage the current board position in FEN format
    const [boardState, setBoardState] = useState(engine.fen());
    // State to manage the board's orientation
    const [boardOrientationState, setBoardOrientationState] = useState<BoardOrientation>('white');
    // State to manage whether the board is locked
    const [isLocked, setIsLocked] = useState(false);
    // State to track if the game is over
    const [gameState, setGameState] = useState(false);
    // State to manage dark mode
    const [isDarkMode, setIsDarkMode] = useState(false);
    // State to track move count
    const [moveCount, setMoveCount] = useState(0);
    // State to track game status
    const [gameStatus, setGameStatus] = useState<'playing' | 'waiting' | 'gameOver'>('playing');

    // Load theme preference from localStorage on component mount
    useEffect(() => {
        const savedTheme = localStorage.getItem('chessTheme');
        if (savedTheme === 'dark') {
            setIsDarkMode(true);
        }
    }, []);

    // Save theme preference to localStorage when it changes
    useEffect(() => {
        localStorage.setItem('chessTheme', isDarkMode ? 'dark' : 'light');
    }, [isDarkMode]);

    // Toggle dark mode
    const toggleDarkMode = () => {
        setIsDarkMode(!isDarkMode);
    };

    // Get current player turn
    const getCurrentPlayer = () => {
        const currentEngine = new Chess();
        currentEngine.load(boardState);
        return currentEngine.turn() === 'w' ? 'White' : 'Black';
    };

    // Get game result message
    const getGameResult = () => {
        const currentEngine = new Chess();
        currentEngine.load(boardState);
        if (currentEngine.isCheckmate()) {
            return `Checkmate! ${currentEngine.turn() === 'w' ? 'Black' : 'White'} wins!`;
        } else if (currentEngine.isDraw()) {
            return 'Game ended in a draw!';
        } else if (currentEngine.isStalemate()) {
            return 'Stalemate! Game is a draw!';
        }
        return 'Game Over!';
    };
    
    // Function to flip the board orientation
    const flipBoard = () => {
        setBoardOrientationState((prev) => (prev === 'white' ? 'black' : 'white'));
    };

    // Function to play a random move on the board
    const playRandomMove = () => {
        // If the board is locked, don't allow a move
        if (isLocked) return;
        
        // Loads the current board state into the chess engine
        const currentEngine = new Chess();
        currentEngine.load(boardState);
        // Gets all legal moves
        const legalMoves = currentEngine.moves({verbose:true});
        // Chooses a random move and keep retrying until we have a valid move
        let randomIndex = Math.floor(Math.random() * legalMoves.length);
        while (true) {
            // eslint-disable-next-line @typescript-eslint/no-use-before-define
            const validator = onDrop(legalMoves[randomIndex].from, legalMoves[randomIndex].to);
            if (validator) break;
            randomIndex = Math.floor(Math.random() * legalMoves.length);
        }        // Checks if game is over
        if(currentEngine.isGameOver()) {
            setIsLocked(true);
            setGameState(true);
            setGameStatus('gameOver');
            // return true; // This return was causing an issue as playRandomMove doesn't return a boolean
        }
    };

    const onDrop = (sourceSquare: string, targetSquare: string): boolean => {
        // If the board is locked, don't allow a move
        if (isLocked) return false;
        
        // Loads the current board state into the chess engine
        const currentEngine = new Chess();
        currentEngine.load(boardState);
        try {
            // Checks if move is valid
            const move = currentEngine.move({
                from: sourceSquare,
                to: targetSquare,
            });
            // If it is valid, update board state, otherwise throw error
            if (move) {
                setBoardState(currentEngine.fen());
                setMoveCount((prev: number) => prev + 1);
                setIsLocked(true);
                setGameStatus('waiting');
                // checks if game is finished
                if(currentEngine.isGameOver()) {
                    setIsLocked(true);
                    setGameState(true);
                    setGameStatus('gameOver');
                    return true;
                }
                // eslint-disable-next-line @typescript-eslint/no-use-before-define
                stockFishMove(currentEngine.fen()).then(() => {
                    // Unlock the board after handling the API response
                    setIsLocked(false);
                    setGameStatus('playing');
                }).catch(error => {
                    console.error('Error in API:', error);
                    // Unlock the board if API error
                    setIsLocked(false);
                    setGameStatus('playing');
                });                // Checks if game is over
                if(currentEngine.isGameOver()){
                    console.log("game won!");
                    setGameState(true);
                    setIsLocked(true);
                    setGameStatus('gameOver');
                }
                return true;
            }
        } catch (error) {
            console.error("Error making move:", error);
        }
        return false;
    };
    // Calls stockfish API and moves a chess piece
    const stockFishMove = async (board: string): Promise<void> => {
        try {
            // Sends the request and throws error if request fails
            const response = await fetch("http://localhost:8081/engine-move", {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    depth: 12,
                    fen: board,
                }),
            });

            if (!response.ok) {
                throw new Error('Network response was not ok');
            }

            const data = await response.json();
            console.log('API response:', data);
            // Extracts values from API response
            const bestMove = data.bestmove;
            const success = data.success;
            
            // Completes the move, updates fen, and checks if game is over
            const currentEngine = new Chess();
            currentEngine.load(board);
            if (bestMove && success) {
                currentEngine.move(bestMove);
                setBoardState(currentEngine.fen());
                setMoveCount((prev: number) => prev + 1);
                if(currentEngine.isGameOver()) {
                    setIsLocked(true);
                    setGameState(true);
                    setGameStatus('gameOver');
                }
            } else {
                throw new Error('API response was not ok');
            }

        } catch (error) {
            console.error('Error calling the API:', error);
            // return; // Removed to avoid unhandled promise rejection
        }
    };    
    // Resets game
    const ResetGame = () => {
        const currentEngine = new Chess(); // Use a local engine instance
        currentEngine.reset();
        setBoardState(currentEngine.fen());
        setIsLocked(false);
        setGameState(false);
        setMoveCount(0);
        setGameStatus('playing');
    };

    return (
        <div className={`${styles.container} ${isDarkMode ? styles.dark : styles.light}`}>
            {/* Theme Toggle Button */}
            <button 
                className={styles.themeToggle}
                onClick={toggleDarkMode}
                title={`Switch to ${isDarkMode ? 'Light' : 'Dark'} Mode`}
            >
                {isDarkMode ? '☀️' : '🌙'}
            </button>

            {/* Header */}
            <div className={styles.header}>
                <h1 className={styles.title}>AI Chess</h1>
            </div>

            {/* Game Info Cards */}
            <div className={styles.gameInfo}>
                <div className={styles.infoCard}>
                    <h4>Current Turn</h4>
                    <p>{getCurrentPlayer()}</p>
                </div>
                <div className={styles.infoCard}>
                    <h4>Moves Played</h4>
                    <p>{moveCount}</p>
                </div>
                <div className={styles.infoCard}>
                    <h4>Game Status</h4>
                    <p>{gameState ? 'Finished' : isLocked ? 'AI Thinking' : 'Active'}</p>
                </div>
            </div>

            {/* Game Status */}
            <div className={`${styles.gameStatus} ${
                gameState ? styles.gameOver : 
                gameStatus === 'waiting' ? styles.waiting : 
                styles.playing
            }`}>
                {gameState ? (
                    <div>
                        🎉 {getGameResult()}
                    </div>
                ) : gameStatus === 'waiting' ? (
                    <div>
                        🤔 AI is thinking...
                    </div>
                ) : (
                    <div>
                        ♟️ Your turn! Make your move.
                    </div>
                )}
            </div>

            {/* Chess Board */}
            <div className={styles.boardContainer}>
                <Chessboard
                    id="BasicBoard"
                    boardOrientation={boardOrientationState}
                    showBoardNotation={true}
                    position={boardState}
                    onPieceDrop={onDrop}
                    boardWidth={500}
                    customBoardStyle={{
                        borderRadius: '12px',
                        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.2)'
                    }}
                    customDarkSquareStyle={{ backgroundColor: '#8B4513' }}
                    customLightSquareStyle={{ backgroundColor: '#F5DEB3' }}
                />
            </div>

            {/* Control Buttons */}
            <div className={styles.controls}>
                <button
                    onClick={playRandomMove}
                    disabled={isLocked || gameState}
                    className={`${styles.button} ${styles.primaryButton}`}
                    title="Play a random move for your side"
                >
                    🎲 Random Move
                </button>
                <button
                    onClick={flipBoard}
                    className={`${styles.button} ${styles.secondaryButton}`}
                    title="Flip the board orientation"
                >
                    🔄 Flip Board
                </button>
                <button
                    onClick={ResetGame}
                    className={`${styles.button} ${styles.dangerButton}`}
                    title="Start a new game"
                >
                    🔄 New Game
                </button>
            </div>
        </div>
    );
}

export default ChessBoard;
